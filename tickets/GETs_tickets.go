package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetTickets(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var tickets []models.Ticket
		if err := app.DB.FindMany(
			ctx,
			ticketsCollection,
			bson.M{"eventid": eventID},
			&tickets,
		); err != nil {
			http.Error(w, "Failed to fetch tickets", http.StatusInternalServerError)
			return
		}

		if tickets == nil {
			tickets = []models.Ticket{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tickets)
	}
}

// Fetch a single ticket
func GetTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		ticketID := ps.ByName("ticketid")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var ticket models.Ticket
		if err := app.DB.FindOne(
			ctx,
			ticketsCollection,
			bson.M{
				"eventid":  eventID,
				"ticketid": ticketID,
			},
			&ticket,
		); err != nil {
			http.Error(w, fmt.Sprintf("Ticket not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticket)
	}
}
