package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"naevis/dels"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// CreateTicket handles creation of a new ticket
func CreateTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		if eventID == "" {
			http.Error(w, "Invalid event ID", http.StatusBadRequest)
			return
		}

		// SECURITY: Verify user is authenticated
		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// SECURITY: Verify user is the event owner
		var event models.Event
		if err := app.DB.FindOne(r.Context(), "events", map[string]interface{}{"eventid": eventID}, &event); err != nil {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}

		if event.CreatorID != userID {
			http.Error(w, "Forbidden: Only event owner can create tickets", http.StatusForbidden)
			return
		}

		var payload struct {
			Name      string  `json:"name"`
			Price     float64 `json:"price"`
			Currency  string  `json:"currency"`
			Quantity  int     `json:"quantity"`
			Color     string  `json:"color"`
			SeatStart int     `json:"seatstart"`
			SeatEnd   int     `json:"seatend"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Validation
		if payload.Name == "" ||
			payload.Currency == "" ||
			payload.Color == "" ||
			payload.Price <= 0 ||
			payload.Quantity < 0 ||
			payload.SeatStart < 0 ||
			payload.SeatEnd < payload.SeatStart {
			http.Error(w, "All fields including seatStart and seatEnd are required", http.StatusBadRequest)
			return
		}

		tick := models.Ticket{
			TicketID:   utils.GenerateRandomString(12),
			EventID:    eventID,
			EntityID:   eventID,
			EntityType: "event",
			Name:       payload.Name,
			Price:      int64(payload.Price * 100), // Convert rupees to paise (int64)
			Currency:   payload.Currency,
			Color:      payload.Color,
			Quantity:   payload.Quantity,
			Available:  payload.Quantity,
			Total:      payload.Quantity,
			SeatStart:  payload.SeatStart,
			SeatEnd:    payload.SeatEnd,
			Sold:       0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := app.DB.Insert(r.Context(), ticketsCollection, tick); err != nil {
			http.Error(w, "Failed to create ticket", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, tick)
	}
}

// EditTicket updates existing ticket fields
func EditTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		ticketID := ps.ByName("ticketid")

		var input models.Ticket
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid input data", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var existing models.Ticket
		if err := app.DB.FindOne(ctx, ticketsCollection, map[string]any{"eventid": eventID, "ticketid": ticketID}, &existing); err != nil {
			http.Error(w, "Ticket not found or DB error", http.StatusNotFound)
			return
		}

		updateFields := map[string]any{}
		if input.Name != "" && input.Name != existing.Name {
			updateFields["name"] = input.Name
		}
		if input.Price > 0 && input.Price != existing.Price {
			updateFields["price"] = input.Price
		}
		if input.Currency != "" && input.Currency != existing.Currency {
			updateFields["currency"] = input.Currency
		}
		if input.Quantity >= 0 && input.Quantity != existing.Quantity {
			updateFields["quantity"] = input.Quantity
			updateFields["available"] = input.Quantity
			updateFields["total"] = input.Quantity
		}
		if input.Color != "" && input.Color != existing.Color {
			updateFields["color"] = input.Color
		}
		if input.SeatStart > 0 && input.SeatStart != existing.SeatStart {
			updateFields["seatstart"] = input.SeatStart
		}
		if input.SeatEnd > 0 && input.SeatEnd != existing.SeatEnd {
			updateFields["seatend"] = input.SeatEnd
		}

		if len(updateFields) == 0 {
			utils.RespondWithJSON(w, http.StatusOK, map[string]any{
				"success": false,
				"message": "No changes detected for ticket",
			})
			return
		}

		updateFields["updated_at"] = time.Now()

		if err := app.DB.UpdateOne(ctx, ticketsCollection, map[string]any{"eventid": eventID, "ticketid": ticketID}, map[string]any{"$set": updateFields}); err != nil {
			http.Error(w, "Failed to update ticket: "+err.Error(), http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Ticket updated successfully",
			"data":    updateFields,
		})
	}
}

// DeleteTicket deletes a ticket via the `dels` package
func DeleteTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		dels.DeleteTicket(app)
	}
}

// BuyTicket purchases a ticket and sets user data
func BuyTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		ticketID := ps.ByName("ticketid")

		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Invalid user", http.StatusBadRequest)
			return
		}

		var body struct {
			Quantity int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quantity <= 0 {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		var ticket models.Ticket
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.FindOne(ctx, ticketsCollection, map[string]any{"eventid": eventID, "ticketid": ticketID}, &ticket); err != nil {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}

		if ticket.Quantity < body.Quantity {
			http.Error(w, "Not enough tickets available", http.StatusBadRequest)
			return
		}

		if err := app.DB.UpdateOne(ctx,
			ticketsCollection,
			map[string]any{"eventid": eventID, "ticketid": ticketID},
			map[string]any{"$inc": map[string]any{"quantity": -body.Quantity, "available": -body.Quantity}},
		); err != nil {
			http.Error(w, "Failed to update ticket quantity", http.StatusInternalServerError)
			return
		}

		m := models.Index{}
		userdata.SetUserData("ticket", ticketID, userID, m.EntityType, m.EntityId, app)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Ticket purchased successfully",
		})
	}
}

// GenerateSeatLabels returns a list of seat labels for a row
func GenerateSeatLabels(start, end int, rowPrefix string) []string {
	var seats []string
	for i := start; i <= end; i++ {
		seats = append(seats, fmt.Sprintf("%s%d", rowPrefix, i))
	}
	return seats
}
