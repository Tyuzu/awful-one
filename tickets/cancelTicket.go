package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"naevis/auditlog"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func CancelTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		requestingUserID := utils.GetUserIDFromRequest(r)

		type CancelPayload struct {
			UniqueCode string `json:"uniqueCode"`
		}

		var payload CancelPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		if payload.UniqueCode == "" {
			http.Error(w, "uniqueCode is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		/* --------------------
		   Fetch purchased ticket
		-------------------- */

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

		/* --------------------
		   Ownership & state checks
		-------------------- */

		if ticket.UserID != requestingUserID {
			http.Error(w, "You are not authorized to cancel this ticket", http.StatusForbidden)
			return
		}

		if ticket.Canceled {
			http.Error(w, "Ticket already canceled", http.StatusBadRequest)
			return
		}

		if ticket.Transferred {
			http.Error(w, "Cannot cancel transferred ticket", http.StatusBadRequest)
			return
		}

		/* --------------------
		   SECURITY: Verify ticket was paid (not free)
		   Only refund if there was an actual payment
		-------------------- */
		if ticket.Price <= 0 {
			// Free ticket - no refund needed
			// Mark as canceled without refund
			if err := app.DB.Update(
				ctx,
				purchasedTicketsCollection,
				bson.M{
					"eventid":    eventID,
					"uniquecode": payload.UniqueCode,
				},
				bson.M{
					"$set": bson.M{
						"canceled":        true,
						"canceledat":      time.Now().UTC(),
						"cancelledreason": "user_requested",
					},
				},
			); err != nil {
				http.Error(w, "Failed to cancel ticket", http.StatusInternalServerError)
				return
			}

			auditlog.LogAction(
				ctx,
				app,
				r,
				requestingUserID,
				models.AuditActionTicketCancel,
				"ticket",
				ticket.TicketID,
				"success",
				map[string]interface{}{
					"reason": "free_ticket_no_refund",
				},
			)

			utils.RespondWithJSON(w, http.StatusOK, map[string]any{
				"success": true,
				"message": "Free ticket canceled successfully",
				"refund":  0,
			})
			return
		}

		/* --------------------
		   Mark ticket as canceled
		-------------------- */

		now := time.Now().UTC()

		update := bson.M{
			"$set": bson.M{
				"canceled":        true,
				"canceledat":      now,
				"cancelledreason": "user_requested",
			},
		}

		if err := app.DB.Update(
			ctx,
			purchasedTicketsCollection,
			bson.M{
				"eventid":    eventID,
				"uniquecode": payload.UniqueCode,
			},
			update,
		); err != nil {
			log.Printf("error canceling ticket: %v", err)
			http.Error(w, "Failed to cancel ticket", http.StatusInternalServerError)
			return
		}

		/* --------------------
		   Create refund request
		-------------------- */

		refund := models.RefundRequest{
			EventID:     eventID,
			TicketID:    ticket.TicketID,
			UserID:      requestingUserID,
			UniqueCode:  ticket.UniqueCode,
			RequestDate: now,
			Status:      "pending",
			Amount:      ticket.Price,
			ProcessedAt: nil,
			RefundedAt:  nil,
		}

		if err := app.DB.Insert(
			ctx,
			refundsCollection,
			refund,
		); err != nil {
			log.Printf("error creating refund record: %v", err)
			http.Error(w, "Ticket canceled, but failed to create refund record", http.StatusInternalServerError)
			return
		}

		/* --------------------
		   Response
		-------------------- */

		// SECURITY: Log audit trail for ticket cancellation and refund
		auditlog.LogAction(
			ctx,
			app,
			r,
			requestingUserID,
			models.AuditActionTicketCancel,
			"ticket",
			ticket.TicketID,
			"success",
			map[string]interface{}{
				"refundAmount": ticket.Price,
				"reason":       "user_requested",
			},
		)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Ticket canceled successfully and marked for refund",
			"refund":  ticket.Price,
		})
	}
}
