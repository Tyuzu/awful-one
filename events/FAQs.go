package events

import (
	"context"
	"encoding/json"
	"log"
	"naevis/infra"
	"naevis/models"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// AddFAQs appends a new FAQ to an event
func AddFAQs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		if eventID == "" {
			log.Println("Missing event ID in request")
			http.Error(w, "Missing event ID", http.StatusBadRequest)
			return
		}

		var newFAQ models.FAQ
		if err := json.NewDecoder(r.Body).Decode(&newFAQ); err != nil {
			log.Printf("Invalid request payload: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if newFAQ.Title == "" || newFAQ.Content == "" {
			http.Error(w, "Title and content are required", http.StatusBadRequest)
			return
		}

		// Use Database interface to add FAQ
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.AddToSet(ctx, eventsCollection, map[string]string{"eventid": eventID}, "faqs", newFAQ); err != nil {
			log.Printf("Error updating event %s: %v", eventID, err)
			http.Error(w, "Error updating event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "FAQ added successfully",
		})
	}
}
