package tickets

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// LockSeats locks specific seats for a user
func LockSeats(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")

		// SECURITY: Get userID from authenticated request context, not from client
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Seats []string `json:"seats"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := map[string]any{
			"eventid":       eventID,
			"seats.seat_id": map[string]any{"$in": req.Seats},
		}
		update := map[string]any{
			"$set": map[string]any{
				"seats.$[].status":  "locked",
				"seats.$[].user_id": userID, // Use authenticated userID, not client-provided
			},
		}

		if err := app.DB.UpdateOne(ctx, ticketsCollection, filter, update); err != nil {
			http.Error(w, `{"error":"Failed to lock seats"}`, http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Seats locked successfully"})
	}
}

// UnlockSeats releases locked seats
func UnlockSeats(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")

		// SECURITY: Get userID from authenticated request context, not from client
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Seats []string `json:"seats"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := map[string]any{
			"eventid":       eventID,
			"seats.seat_id": map[string]any{"$in": req.Seats},
			"seats.user_id": userID, // Only unlock seats locked by this user
		}
		update := map[string]any{
			"$set": map[string]any{
				"seats.$[].status":  "available",
				"seats.$[].user_id": nil,
			},
		}

		if err := app.DB.UpdateOne(ctx, ticketsCollection, filter, update); err != nil {
			http.Error(w, `{"error":"Failed to unlock seats"}`, http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Seats unlocked successfully"})
	}
}

// ConfirmSeatPurchase marks locked seats as booked
func ConfirmSeatPurchase(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		ticketID := ps.ByName("ticketid")

		// SECURITY: Get userID from authenticated request context, not from client
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Seats []string `json:"seats"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var ticket models.Ticket
		if err := app.DB.FindOne(ctx, ticketsCollection, map[string]any{"ticketid": ticketID, "eventid": eventID}, &ticket); err != nil {
			http.Error(w, `{"error":"Ticket not found"}`, http.StatusNotFound)
			return
		}

		// Verify all requested seats are locked to this user
		for _, seat := range ticket.Seats {
			for _, id := range req.Seats {
				if seat.SeatID == id && (seat.Status != "locked" || seat.UserID != userID) {
					http.Error(w, `{"error":"Some seats are not properly locked or not locked to you"}`, http.StatusConflict)
					return
				}
			}
		}

		update := map[string]any{"$set": map[string]any{"seats.$[].status": "booked"}}
		if err := app.DB.UpdateOne(ctx, ticketsCollection, map[string]any{"ticketid": ticketID, "eventid": eventID, "seats.seat_id": map[string]any{"$in": req.Seats}}, update); err != nil {
			http.Error(w, `{"error":"Failed to confirm purchase"}`, http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Ticket purchased successfully"})
	}
}
