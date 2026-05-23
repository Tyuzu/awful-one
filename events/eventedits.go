package events

import (
	"context"
	"log"
	"naevis/dels"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// EditEvent updates an existing event
func EditEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		if eventID == "" {
			http.Error(w, "Missing event ID", http.StatusBadRequest)
			return
		}

		updateFields, err := updateEventFields(r)
		if err != nil {
			log.Printf("Invalid update fields for event %s: %v", eventID, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := validateUpdateFields(updateFields); err != nil {
			log.Printf("Validation failed for event %s: %v", eventID, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		updateFields["updated_at"] = time.Now().UTC()

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Update using Database interface
		if err := app.DB.UpdateOne(
			ctx,
			eventsCollection,
			map[string]string{"eventid": eventID},
			map[string]any{"$set": updateFields},
		); err != nil {
			log.Printf("Error updating event %s: %v", eventID, err)
			http.Error(w, "Error updating event", http.StatusInternalServerError)
			return
		}

		// Fetch updated event
		var updatedEvent models.Event
		if err := app.DB.FindOne(
			ctx,
			eventsCollection,
			map[string]string{"eventid": eventID},
			&updatedEvent,
		); err != nil {
			http.Error(w, "Error retrieving updated event", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, updatedEvent)
	}
}

// DeleteEvent deletes an event and its related data
func DeleteEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		dels.DeleteEvent(app)
	}
}
