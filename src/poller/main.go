// main.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
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

func put_response(locationId int) (int, error) {
	return 0, nil
}

func handler() (string, error) {

	appt, err := get_avail_slots(8120)
	if err != nil {
		fmt.Println("error")
	}

	locationData := &LocationState{
		LocationId:   8120,
		Availability: appt,
	}

	resp, err := json.Marshal(locationData)
	if err != nil {
		panic(err)
	}

	return string(resp), nil
}

func main() {
	lambda.Start(handler)
}
