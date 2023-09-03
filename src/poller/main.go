// main.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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
	if err := json.Unmarshal(responseBody, data); err != nil {
		fmt.Println("error unmarshalling JSON")
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
	ddb := createDynamoSession()
	stateMap, marshalErr := dynamodbattribute.MarshalMap(
		&struct {
			locationId int
			state      LocationState
		}{
			locationId: s.LocationId,
			state:      *s.CurrentState,
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

	_, writeErr := ddb.PutItem(input)
	if writeErr != nil {
		fmt.Println("Failed to write to DDB table")
		return writeErr
	}

	return nil
}

func (s *Location) getPreviousState() error {
	session := createDynamoSession()
	result, err := session.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"locationId": &types.AttributeValueMemberS{Value: string(s.LocationId)},
		},
	})
	if err != nil {
		fmt.Println("Error getting location")
		return err
	}
	if result.Item == nil {
		msg := "Could not get prev state for location '" + string(s.LocationId) + "'"
		return errors.New(msg)
	}
	err = dynamodbattribute.UnmarshalMap(result.Item.state, s.PreviousState)
	if err != nil {
		msg := "Could not unmarshall for location '" + string(s.LocationId) + "'"
		return errors.New(msg)
	}

	return nil
}

func (s Location) compare() error {

	return nil
}

func createDynamoSession() *dynamodb.DynamoDB {
	sesh := session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
		},
	))

	return dynamodb.New(sesh)
}

func get_locations() {
	//TODO Implement
	return
}

func handler() (string, error) {
	boise := newLocation(12161)
	err := boise.getCurrentState()
	fmt.Println(boise.CurrentState.NextAppointmentDate)
	if err != nil {
		fmt.Println("Could not read avail slots for location %d", boise.LocationId)
		fmt.Println(err)
		return "FAIL", err
	}
	err = boise.storeCurrentState()
	if err != nil {
		fmt.Println("Could not write state to DDB for loc %d", boise.LocationId)
		fmt.Println(err)
		return "FAIL", err
	}

	return "Run OK", nil
}

func main() {
	lambda.Start(handler)
}
