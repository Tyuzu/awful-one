package events

import (
	"context"
	"encoding/json"
	"log"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

// CreateEvent handles creating a new event
func CreateEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		event, err := parseEventData(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		requestingUserID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Invalid user", http.StatusBadRequest)
			return
		}

		prepareEventDefaults(&event, requestingUserID, app)

		if err := parseArtistData(r, &event); err != nil {
			http.Error(w, "Invalid artists data", http.StatusBadRequest)
			return
		}

		// Insert to DB using the Database interface
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.Insert(ctx, eventsCollection, event); err != nil {
			log.Printf("DB insert error: %v", err)
			http.Error(w, "Error saving event", http.StatusInternalServerError)
			return
		}

		userdata.SetUserData("event", event.EventID, requestingUserID, "", "", app)

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(event); err != nil {
			log.Printf("Encoding response error: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func prepareEventDefaults(event *models.Event, userID string, app *infra.Deps) {
	event.CreatorID = userID
	event.CreatedAt = time.Now().UTC()
	event.Date = event.Date.UTC()
	event.Status = "active"
	event.FAQs = []models.FAQ{}
	event.Artists = []string{}
	event.Tags = []string{}
	event.Merch = []models.Merch{}
	event.Tickets = []models.Ticket{}
	event.OrganizerName = strings.TrimSpace(event.OrganizerName)
	event.OrganizerContact = strings.TrimSpace(event.OrganizerContact)

	event.EventID = utils.GenerateRandomString(14)

	// Ensure no collision using Database interface
	var existingEvent models.Event
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := app.DB.FindOne(ctx, eventsCollection, map[string]string{"eventid": event.EventID}, &existingEvent); err == nil {
		event.EventID = utils.GenerateRandomString(14) // regenerate once
	}
}

func parseArtistData(r *http.Request, event *models.Event) error {
	artistStr := r.FormValue("artists")
	if artistStr == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(artistStr), &ids); err != nil {
		return err
	}
	event.Artists = ids
	return nil
}

func parseEventData(r *http.Request) (models.Event, error) {
	var event models.Event
	data := r.FormValue("event")
	if data == "" {
		return event, http.ErrMissingFile
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return event, err
	}
	return event, nil
}
