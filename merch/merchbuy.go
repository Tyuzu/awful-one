package merch

import (
	"encoding/json"
	"log"
	"net/http"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/stripe"
	"naevis/userdata"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// POST /merch/event/:eventId/:merchId/payment-session
func CreateMerchPaymentSession(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		merchID := ps.ByName("merchid")
		eventID := ps.ByName("eventid")

		var body struct {
			Quantity int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quantity < 1 {
			http.Error(w, "Invalid request or quantity", http.StatusBadRequest)
			return
		}

		session, err := stripe.CreateMerchSession(merchID, eventID, body.Quantity)
		if err != nil {
			log.Printf("Error creating payment session: %v", err)
			http.Error(w, "Failed to create payment session", http.StatusInternalServerError)
			return
		}

		response := map[string]any{
			"success": true,
			"data": map[string]any{
				"paymentUrl": session.URL,
				"eventId":    session.EventID,
				"merchId":    session.MerchID,
				"quantity":   session.Stock,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}

// POST /merch/event/:eventId/:merchId/confirm-purchase
func ConfirmMerchPurchase(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		var body struct {
			Quantity int     `json:"quantity"`
			Price    float64 `json:"price,omitempty"` // Optional, will be verified against DB
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quantity < 1 {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		eventID := ps.ByName("eventid")
		merchID := ps.ByName("merchid")

		userID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Invalid user", http.StatusUnauthorized)
			return
		}

		// SECURITY: First lookup current merch details to verify price
		var currentMerch models.Merch
		if err := app.DB.FindOne(
			ctx,
			merchCollection,
			bson.M{
				"entity_id": eventID,
				"merchid":   merchID,
			},
			&currentMerch,
		); err != nil {
			http.Error(w, "Merch not found", http.StatusNotFound)
			return
		}

		// Verify price matches current database price (with small tolerance for floating point)
		if body.Price > 0 {
			priceDiff := currentMerch.Price - body.Price
			if priceDiff < -0.01 || priceDiff > 0.01 { // Allow 1 paise tolerance
				http.Error(w, "Price mismatch. Please refresh and try again", http.StatusConflict)
				return
			}
		}

		// Check stock availability
		if currentMerch.Stock < body.Quantity {
			http.Error(w, "Not enough merch available", http.StatusBadRequest)
			return
		}

		// Atomically decrement stock and return updated document
		var updatedMerch models.Merch
		err := app.DB.FindOneAndUpdate(
			ctx,
			merchCollection,
			bson.M{
				"entity_id": eventID,
				"merchid":   merchID,
				"stock":     bson.M{"$gte": body.Quantity},
			},
			bson.M{
				"$inc": bson.M{"stock": -body.Quantity},
			},
			&updatedMerch,
		)
		if err != nil {
			http.Error(w, "Not enough merch available", http.StatusBadRequest)
			return
		}

		// Record purchase
		userdata.SetUserData(
			"merch",
			updatedMerch.MerchID,
			userID,
			updatedMerch.EntityType,
			updatedMerch.EntityID,
			app,
		)

		resp := map[string]any{
			"success": true,
			"data": map[string]any{
				"message":        "Merch purchased successfully",
				"merchId":        updatedMerch.MerchID,
				"eventId":        updatedMerch.EntityID,
				"quantityBought": body.Quantity,
				"remainingStock": updatedMerch.Stock,
				"unitPrice":      currentMerch.Price,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
