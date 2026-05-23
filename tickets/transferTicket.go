package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func TransferTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		requestingUserId := utils.GetUserIDFromRequest(r)

		type TransferPayload struct {
			UniqueCode string `json:"uniqueCode"`
			Recipient  string `json:"recipient"`
		}

		var payload TransferPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		if payload.UniqueCode == "" || payload.Recipient == "" {
			http.Error(w, "uniqueCode and recipient are required", http.StatusBadRequest)
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
				"uniquecode": payload.UniqueCode,
			},
			&ticket,
		); err != nil {
			http.Error(w, fmt.Sprintf("Ticket not found: %v", err), http.StatusNotFound)
			return
		}

		// Ownership check
		if ticket.UserID != requestingUserId {
			http.Error(w, "You are not authorized to transfer this ticket", http.StatusForbidden)
			return
		}

		// Update ownership
		if err := app.DB.UpdateOne(
			ctx,
			purchasedTicketsCollection,
			bson.M{
				"eventid":    eventID,
				"uniquecode": payload.UniqueCode,
			},
			bson.M{
				"$set": bson.M{
					"userid":        payload.Recipient,
					"transferred":   true,
					"transferredto": payload.Recipient,
				},
			},
		); err != nil {
			http.Error(w, fmt.Sprintf("Failed to transfer ticket: %v", err), http.StatusInternalServerError)
			return
		}

		// Update user data indexes
		userdata.DelUserData("ticket", payload.UniqueCode, requestingUserId, app)
		userdata.SetUserData("ticket", payload.UniqueCode, payload.Recipient, "event", eventID, app)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Ticket transferred successfully",
		})
	}
}
