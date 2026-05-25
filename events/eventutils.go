package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func updateEventFields(r *http.Request) (bson.M, error) {
	// Parse the multipart form with a 10MB limit
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return nil, fmt.Errorf("unable to parse form: %v", err)
	}

	updateFields := bson.M{}

	// Extract "event" field from form-data
	eventJSON := r.FormValue("event")
	if eventJSON == "" {
		return nil, fmt.Errorf("missing event data")
	}

	// Define a struct to parse the JSON
	var eventData struct {
		Title       string `json:"title"`
		Date        string `json:"date"`
		Category    string `json:"category"`
		Location    string `json:"location"`
		PlaceId     string `json:"placeid"`
		PlaceName   string `json:"placename"`
		Description string `json:"description"`
	}

	// Decode the JSON
	if err := json.Unmarshal([]byte(eventJSON), &eventData); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %v", err)
	}

	// Map the fields to updateFields
	if eventData.Category != "" {
		updateFields["category"] = eventData.Category
	}
	if eventData.Title != "" {
		updateFields["title"] = eventData.Title
	}
	if eventData.Date != "" {
		parsedDateTime, err := time.Parse(time.RFC3339, eventData.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, expected RFC3339 (YYYY-MM-DDTHH:MM:SSZ)")
		}
		updateFields["date"] = parsedDateTime.UTC()
	}
	if eventData.Location != "" {
		updateFields["location"] = eventData.Location
	}
	if eventData.PlaceId != "" {
		updateFields["placeid"] = eventData.PlaceId
	}
	if eventData.PlaceName != "" {
		updateFields["placename"] = eventData.PlaceName
	}
	if eventData.Description != "" {
		updateFields["description"] = eventData.Description
	}

	return updateFields, nil
}

// Validate required fields
func validateUpdateFields(updateFields bson.M) error {
	if updateFields["category"] == "" || updateFields["title"] == "" || updateFields["location"] == "" || updateFields["description"] == "" {
		return fmt.Errorf("category, title, location, and description are required")
	}
	return nil
}
