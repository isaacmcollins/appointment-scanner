// main.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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
	LocationId   int
	Availability *ApiResponse
}

const tableName string = "state-store"
const baseUrl string = "https://ttp.cbp.dhs.gov/schedulerapi"

func newLocation(locationId int) *LocationState {
	locationState := LocationState{
		LocationId:   locationId,
		Availability: nil,
	}
	return &locationState
}

func (s *LocationState) getCurrentSlots() error {
	api := fmt.Sprintf("%s/slot-availability?locationId=%d", baseUrl, s.LocationId)
	response, err := http.Get(api)
	if err != nil {
		fmt.Println("Could not get response from API")
		return err
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Could not read response body")
	}

	if err := json.Unmarshal(responseBody, s.Availability); err != nil {
		fmt.Println("error unmarshalling JSON")
		return err
	}

	return err
}

func (s LocationState) storeState() error {
	ddb := createDynamoSession()

	stateMap, marshalErr := dynamodbattribute.MarshalMap(s)
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

func (s *LocationState) getLastState() error {
	session := createDynamoSession()
	result, err := session.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"locationId": &types.AttributeValueMemberS{Value: string(s.LocationId)},
		},
	})
	if err != nil {
		fmt.Println("Error getting location")
		return nil, err
	}
	if result.Item == nil {
		msg := "Could not get location '" + string(s.LocationId) + "'"
		return errors.New(msg)
	}
	err = dynamodbattribute.UnmarshalMap(result.Item, s.Availability)
	if err != nil {
		msg := "Could not unmarshall for location '" + string(s.LocationId) + "'"
		return errors.New(msg)
	}

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
	err := boise.getCurrentSlots()
	if err != nil {
		fmt.Println("Could not read avail slots for location %d", boise.LocationId)
		fmt.Println(err)
		return "FAIL", err
	}
	err = boise.storeState()
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
