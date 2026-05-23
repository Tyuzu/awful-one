package tickets

import (
	"context"
	"encoding/json"
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

func BuysTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		type Request struct {
			TicketID string `json:"ticketId"`
			EventID  string `json:"eventId"`
			Quantity int    `json:"quantity"`
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.TicketID == "" || req.EventID == "" || req.Quantity <= 0 {
			http.Error(w, "Missing or invalid parameters", http.StatusBadRequest)
			return
		}

		// SECURITY: Extract and verify user authentication
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		/* --------------------
		   Atomic ticket purchase with race condition fix
		   Uses FindOneAndUpdate to ensure atomicity
		-------------------- */

		var ticket models.Ticket

		err := app.DB.FindOneAndUpdate(
			ctx,
			ticketsCollection,
			bson.M{
				"ticketid":  req.TicketID,
				"eventid":   req.EventID,
				"available": bson.M{"$gte": req.Quantity}, // Atomic check: only update if enough tickets
			},
			bson.M{
				"$inc": bson.M{
					"available": -req.Quantity,
					"sold":      req.Quantity,
				},
				"$set": bson.M{
					"updatedat": time.Now().UTC(),
				},
			},
			&ticket,
		)

		if err != nil {
			http.Error(w, "Ticket not found or insufficient quantity available", http.StatusBadRequest)
			return
		}

		/* --------------------
		   Create booking record with userID
		-------------------- */

		booking := Ticking{
			BookingID: utils.GenerateRandomString(14),
			EventID:   req.EventID,
			TicketID:  req.TicketID,
			UserID:    userID, // SECURITY: Now tracks which user made the purchase
			Quantity:  req.Quantity,
			BookedAt:  time.Now().UTC(),
		}

		if err := app.DB.Insert(
			ctx,
			bookingsCollection,
			booking,
		); err != nil {
			log.Println("warning: booking insert failed:", err)
		}

		/* --------------------
		   Track purchased ticket with userID
		-------------------- */

		if err := app.DB.Insert(
			ctx,
			purchasedTicketsCollection,
			bson.M{
				"ticketid":  req.TicketID,
				"eventid":   req.EventID,
				"userid":    userID, // SECURITY: Tracks user who purchased
				"quantity":  req.Quantity,
				"purchased": time.Now().UTC(),
			},
		); err != nil {
			log.Println("warning: purchased ticket insert failed:", err)
		}

		/* --------------------
		   Response
		-------------------- */

		// SECURITY: Log audit trail for ticket purchase
		auditlog.LogAction(
			ctx,
			app,
			r,
			userID,
			models.AuditActionTicketPurchase,
			"ticket",
			req.TicketID,
			"success",
			map[string]interface{}{
				"eventId":  req.EventID,
				"quantity": req.Quantity,
			},
		)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Ticket booked successfully",
		})
	}
}

/* --------------------
   Booking Model
-------------------- */

type Ticking struct {
	BookingID string    `bson:"bookingid"`
	EventID   string    `bson:"eventid"`
	TicketID  string    `bson:"ticketid"`
	UserID    string    `bson:"userid"`
	Quantity  int       `bson:"quantity"`
	BookedAt  time.Time `bson:"bookedat"`
}
