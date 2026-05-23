package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"naevis/infra"
	"naevis/models"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func VerifyTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		uniqueCode := r.URL.Query().Get("uniqueCode")

		if uniqueCode == "" {
			http.Error(w, "Unique code is required for verification", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var ticket models.PurchasedTicket
		if err := app.DB.FindOne(
			ctx,
			purchasedTicketsCollection,
			bson.M{
				"eventid":    eventID,
				"uniquecode": uniqueCode,
			},
			&ticket,
		); err != nil {
			http.Error(w, fmt.Sprintf("Ticket verification failed: %v", err), http.StatusNotFound)
			return
		}

		// SECURITY: Check if ticket has been canceled
		if ticket.Canceled {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"isValid":      false,
				"reason":       "Ticket has been canceled",
				"canceledAt":   ticket.CanceledAt,
				"cancelReason": ticket.CanceledReason,
			})
			return
		}

		// SECURITY: Check if ticket has been transferred
		if ticket.Transferred {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"isValid":       false,
				"reason":        "Ticket has been transferred",
				"transferredTo": ticket.TransferredTo,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{
			"isValid": true,
		})
	}
}
