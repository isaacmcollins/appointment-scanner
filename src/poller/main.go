// main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

type ApiResponse struct {
	AvailableSlots []struct {
		LocationID     int    `json:"locationId"`
		StartTimestamp string `json:"startTimestamp"`
		EndTimestamp   string `json:"endTimestamp"`
		Active         bool   `json:"active"`
		Duration       int    `json:"duration"`
		RemoteInd      bool   `json:"remoteInd"`
	} `json:"availableSlots"`
	LastPublishedDate string `json:"lastPublishedDate"`
}

type LocationState struct {
	NextAppointmentDate time.Time
	Active              bool
}

type Location struct {
	LocationId    int
	PreviousState *LocationState
	CurrentState  *LocationState
}

var logger zerolog.Logger

const tableName string = "state-store"
const baseUrl string = "https://ttp.cbp.dhs.gov/schedulerapi"

func newLocation(locationId int) *Location {
	locationState := Location{
		LocationId:    locationId,
		PreviousState: nil,
		CurrentState:  nil,
	}
	return &locationState
}

func (s *Location) getCurrentState() error {
	api := fmt.Sprintf("%s/slot-availability?locationId=%d", baseUrl, s.LocationId)
	response, err := http.Get(api)
	if err != nil {
		fmt.Println("Could not get response from API")
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Could not read response body")
	}

	data := ApiResponse{}
	if err := json.Unmarshal(responseBody, &data); err != nil {
		fmt.Println("error unmarshalling JSON")
		fmt.Println(err)
	}

	timestamp, err := time.Parse("2006-01-02T15:05", data.AvailableSlots[0].StartTimestamp)
	if err != nil {
		fmt.Println("Could not pars timestamp %s", data.AvailableSlots[0].StartTimestamp)
		return err
	}

	s.CurrentState = &LocationState{
		NextAppointmentDate: timestamp,
		Active:              data.AvailableSlots[0].Active,
	}
	return err
}

func (s Location) storeCurrentState() error {
	ctx := context.Background()
	ddb := createDynamoSession()
	stateMap, marshalErr := attributevalue.MarshalMap(
		&struct {
			LocationId int
			State      LocationState
		}{
			LocationId: s.LocationId,
			State:      *s.CurrentState,
		},
	)
	if marshalErr != nil {
		fmt.Println("Failed to marshal to dynamo map")
		return marshalErr
	}

	input := &dynamodb.PutItemInput{
		Item:      stateMap,
		TableName: aws.String(tableName),
	}

	out, writeErr := ddb.PutItem(ctx, input)
	if writeErr != nil {
		fmt.Println("Failed to write to DDB table")
		return writeErr
	}
	fmt.Println(out)

	return nil
}

func (s *Location) getPreviousState() error {
	ctx := context.Background()
	session := createDynamoSession()
	result, err := session.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"LocationId": &types.AttributeValueMemberN{Value: strconv.Itoa(s.LocationId)},
		},
	})
	if err != nil {
		fmt.Println("Error getting location")
		return err
	}
	if result.Item == nil {
		msg := "Could not get prev state for location '" + strconv.Itoa(s.LocationId) + "'"
		return errors.New(msg)
	}
	err = attributevalue.Unmarshal(result.Item["State"], &s.PreviousState)
	if err != nil {
		msg := "Could not unmarshall for location '" + strconv.Itoa(s.LocationId) + "'"
		return errors.New(msg)
	}

	return nil
}

func (s Location) isNewAppointmentDate() bool {
	if s.CurrentState.NextAppointmentDate.Before(
		s.PreviousState.NextAppointmentDate) {
		return true
	}
	return false
}

func createDynamoSession() *dynamodb.Client {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}

	return dynamodb.NewFromConfig(cfg)
}

func get_locations() {
	//TODO Implement
	return
}

func handler() (string, error) {
	boise := newLocation(12161)
	err := boise.getCurrentState()
	if err != nil {
		log.Println("{ :", boise.LocationId)
		log.Println(err)
		return "FAIL", err
	}
	err = boise.storeCurrentState()
	if err != nil {
		fmt.Println("Could not write state to DDB for loc %d", boise.LocationId)
		fmt.Println(err)
		return "FAIL", err
	}
	err = boise.getPreviousState()
	if err != nil {
		fmt.Println("Could retreive previous state from DDB for loc %d", boise.LocationId)
		fmt.Println(err)
		return "FAIL", err
	}

	if boise.isNewAppointmentDate() {
		res := fmt.Sprintf("NEW SLOT %s", boise.CurrentState.NextAppointmentDate)
		logger.Info().Msg(res)
		return res, nil
	}

	return "OK", nil
}

func main() {
	logger = zlog.With().
		Str("function", "poller").
		Logger()
	lambda.Start(handler)
}
