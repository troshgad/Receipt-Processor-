package main

/*

 */
import (
	"encoding/json"

	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}
type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ProcessResponse struct {
	Id string `json:"id"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

var receipts = make(map[string]Receipt)

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/receipts/process", ProcessHandler).Methods("POST")
	r.HandleFunc("/receipts/{id}/points", PointsHandler).Methods("GET")
	http.Handle("/", r)
	http.ListenAndServe(":80", nil)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {

}

/*
Endpoint: Process Receipts
Path: /receipts/process
Method: POST
Payload: Receipt JSON
Response: JSON containing an id for the receipt.
Description:

Takes in a JSON receipt (see example in the example directory) and returns a JSON object with an ID generated by your code.

The ID returned is the ID that should be passed into /receipts/{id}/points to get the number of points the receipt was awarded.

How many points should be earned are defined by the rules below.

Reminder: Data does not need to survive an application restart. This is to allow you to use in-memory solutions to track any data generated by this endpoint.

Example Response:

{ "id": "7fb1377b-b223-49d9-a31a-5a02701dd310" }
*/
func ProcessHandler(w http.ResponseWriter, r *http.Request) {
	//var receipt Receipt
	var receipt Receipt

	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {

		http.Error(w, "the receipt is invalid", http.StatusBadRequest)
		return
	} else if validate_request(receipt) {
		var idString = uuid.New().String()
		response := ProcessResponse{
			Id: idString,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		receipts[idString] = receipt

	} else {
		http.Error(w, "the receipt is invalid", http.StatusBadRequest)
	}

}

/*
Endpoint: Get Points
Path: /receipts/{id}/points
Method: GET
Response: A JSON object containing the number of points awarded.
A simple Getter endpoint that looks up the receipt by the ID and returns an object specifying the points awarded.

Example Response:

{ "points": 32 }
Rules
These rules collectively define how many points should be awarded to a receipt.

One point for every alphanumeric character in the retailer name.
50 points if the total is a round dollar amount with no cents.
25 points if the total is a multiple of 0.25.
5 points for every two items on the receipt.
If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned.
6 points if the day in the purchase date is odd.
10 points if the time of purchase is after 2:00pm and before 4:00pm.
Examples

	{
	  "retailer": "Target",
	  "purchaseDate": "2022-01-01",
	  "purchaseTime": "13:01",
	  "items": [
	    {
	      "shortDescription": "Mountain Dew 12PK",
	      "price": "6.49"
	    },{
	      "shortDescription": "Emils Cheese Pizza",
	      "price": "12.25"
	    },{
	      "shortDescription": "Knorr Creamy Chicken",
	      "price": "1.26"
	    },{
	      "shortDescription": "Doritos Nacho Cheese",
	      "price": "3.35"
	    },{
	      "shortDescription": "   Klarbrunn 12-PK 12 FL OZ  ",
	      "price": "12.00"
	    }
	  ],
	  "total": "35.35"
	}

Total Points: 28
Breakdown:

	   6 points - retailer name has 6 characters
	  10 points - 5 items (2 pairs @ 5 points each)
	   3 Points - "Emils Cheese Pizza" is 18 characters (a multiple of 3)
	              item price of 12.25 * 0.2 = 2.45, rounded up is 3 points
	   3 Points - "Klarbrunn 12-PK 12 FL OZ" is 24 characters (a multiple of 3)
	              item price of 12.00 * 0.2 = 2.4, rounded up is 3 points
	   6 points - purchase day is odd
	+ ---------
	= 28 points

	{
	  "retailer": "M&M Corner Market",
	  "purchaseDate": "2022-03-20",
	  "purchaseTime": "14:33",
	  "items": [
	    {
	      "shortDescription": "Gatorade",
	      "price": "2.25"
	    },{
	      "shortDescription": "Gatorade",
	      "price": "2.25"
	    },{
	      "shortDescription": "Gatorade",
	      "price": "2.25"
	    },{
	      "shortDescription": "Gatorade",
	      "price": "2.25"
	    }
	  ],
	  "total": "9.00"
	}

Total Points: 109
Breakdown:

	  50 points - total is a round dollar amount
	  25 points - total is a multiple of 0.25
	  14 points - retailer name (M&M Corner Market) has 14 alphanumeric characters
	              note: '&' is not alphanumeric
	  10 points - 2:33pm is between 2:00pm and 4:00pm
	  10 points - 4 items (2 pairs @ 5 points each)
	+ ---------
	= 109 points
*/

/*
One point for every alphanumeric character in the retailer name.
50 points if the total is a round dollar amount with no cents.
25 points if the total is a multiple of 0.25.
5 points for every two items on the receipt.
If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned.
6 points if the day in the purchase date is odd.
10 points if the time of purchase is after 2:00pm and before 4:00pm.
*/
func PointsHandler(w http.ResponseWriter, r *http.Request) {
	parameters := mux.Vars(r)
	points := 0
	id := parameters["id"]
	if receipts[id].Retailer == "" {
		http.Error(w, "No receipt found for that id ", http.StatusNotFound)
		return
	} else {

		receipt := receipts[id]
		retailertWithoutSymbols := strings.Join(strings.Split(regexp.MustCompile("[^a-zA-Z0-9 ]+").ReplaceAllString(receipt.Retailer, ""), " "), "")

		//One point for every alphanumeric character in the retailer name.
		points += len(retailertWithoutSymbols)

		//50 points if the total is a round dollar amount with no cents.
		if strings.Split(receipt.Total, ".")[1] == "00" {

			points += 50
		}

		//25 points if the total is a multiple of 0.25.
		totalCents := strings.Split(receipt.Total, ".")[1]
		if totalCents == "25" || totalCents == "00" || totalCents == "50" || totalCents == "75" {

			points += 25
		}

		//5 points for every two items on the receipt.
		points += (5 * (int(len(receipt.Items) / 2)))

		//If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned.
		itemDescriptionPoints := 0.0
		for _, a := range receipt.Items {
			trimmedItemDescription := strings.TrimSpace(a.ShortDescription)

			if len(trimmedItemDescription)%3 == 0 {
				priceAsAFloat, priceErr := strconv.ParseFloat(a.Price, 64)
				if priceErr != nil {
					http.Error(w, "Conversion Failure", http.StatusBadRequest)
					return
				} else {
					itemPoints := math.Ceil(priceAsAFloat * .2)
					itemDescriptionPoints += itemPoints

				}

			}
		}

		points += int(itemDescriptionPoints)

		//6 points if the day in the purchase date is odd.
		//Didnt want to do a conversion when the dates have to be formatted a certain way anyways
		if receipt.PurchaseDate[9] == '1' || receipt.PurchaseDate[9] == '3' || receipt.PurchaseDate[9] == '5' || receipt.PurchaseDate[9] == '7' || receipt.PurchaseDate[9] == '9' {
			points += 6

		}

		//10 points if the time of purchase is after 2:00pm and before 4:00pm.
		purchaseTimeHours, timeHourError := strconv.ParseInt(strings.Split(receipt.PurchaseTime, ":")[0], 10, 16)
		purchaseTimeMinute, timeMinuteError := strconv.ParseInt(strings.Split(receipt.PurchaseTime, ":")[1], 10, 16)
		if timeHourError != nil || timeMinuteError != nil {
			http.Error(w, "Conversion Failure", http.StatusBadRequest)
			return
		}
		if (purchaseTimeHours >= 14 && purchaseTimeMinute > 1) && purchaseTimeHours < 16 {

			points += 10
		}

		response := PointsResponse{
			Points: points,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
