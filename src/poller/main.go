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

func get_avail_slots(locationId int) (*ApiResponse, error) {
	api := fmt.Sprintf("https://ttp.cbp.dhs.gov/schedulerapi/slot-availability?locationId=%d", locationId)
	var result ApiResponse
	response, err := http.Get(api)
	if err != nil {
		fmt.Println("Could not get response from API")
		return &result, err
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Could not read response body")
	}

	if err := json.Unmarshal(responseBody, &result); err != nil {
		fmt.Println("error unmarshalling JSON")
		return &result, err
	}

	return &result, err
}

func update_state(locationId int, state *LocationState) error {
	ddb := createDynamoSession()

	stateMap, marshalErr := dynamodbattribute.MarshalMap(&state)
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

func getState(locationId int) (*LocationState, error) {
	session := createDynamoSession()

	result, err := session.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"locationId": &types.AttributeValueMemberS{Value: locationId},
		},
	})
	if err != nil {
		fmt.Println("Error getting location")
		return err
	}
	if result.Item == nil {
		msg := "Could not get location '" + string(locationId) + "'"
		return nil, errors.New(msg)
	}

	location := LocationState{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &location)
	if err != nil {
		msg := "Could not unmarshall for location '" + string(locationId) + "'"
		return nil, errors.New(msg)
	}

	return &location, nil
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
	return
}

func handler() (string, error) {
	appt, err := get_avail_slots(12161)
	if err != nil {
		fmt.Println("Polling error")
	}
	locationData := &LocationState{ //redundant
		LocationId:   12161,
		Availability: appt,
	}

	err = update_state(12161, locationData)
	if err != nil {
		fmt.Println("Error writing to dynamodb")
		fmt.Println(err)
	}

	return "OK", nil
}

func main() {
	lambda.Start(handler)
}
