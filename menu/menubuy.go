package menu

import (
	"context"
	"encoding/json"
	"log"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/stripe"
	"naevis/userdata"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// MenuPurchaseRequest represents the request body for purchasing menus
type MenuPurchaseRequest struct {
	MenuID  string `json:"menuId"`
	PlaceId string `json:"placeId"`
	Stock   int    `json:"quantity"`
}

// MenuPurchaseResponse represents the response body for menu purchase confirmation
type MenuPurchaseResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// ConfirmMenuPurchase handles the POST request for confirming the menu purchase
func ConfirmMenuPurchase(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var request MenuPurchaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		requestingUserID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || requestingUserID == "" {
			http.Error(w, "Invalid user", http.StatusUnauthorized)
			return
		}

		request.PlaceId = ps.ByName("placeid")
		request.MenuID = ps.ByName("menuid")

		buyMenu(w, request, requestingUserID, app)
	}
}

// buyMenu handles the actual menu purchase
func buyMenu(w http.ResponseWriter, request MenuPurchaseRequest, requestingUserID string, app *infra.Deps) {
	placeId := request.PlaceId
	menuID := request.MenuID
	stockRequested := request.Stock

	// Fetch the menu using Database interface
	var menu models.Menu
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := app.DB.FindOne(ctx, menuCollection, map[string]string{
		"placeid": placeId,
		"menuid":  menuID,
	}, &menu)
	if err != nil {
		http.Error(w, "Menu not found", http.StatusNotFound)
		return
	}

	// Check stock availability
	if menu.Stock < stockRequested {
		http.Error(w, "Not enough menu available", http.StatusBadRequest)
		return
	}

	// Update stock in database
	update := map[string]any{"$inc": map[string]int{"stock": -stockRequested}}
	if err := app.DB.UpdateOne(ctx, menuCollection, map[string]string{"placeid": placeId, "menuid": menuID}, update); err != nil {
		http.Error(w, "Failed to update menu stock", http.StatusInternalServerError)
		return
	}

	// Save user purchase data
	userdata.SetUserData("menu", menuID, requestingUserID, "place", placeId, app)

	// Respond with success
	response := MenuPurchaseResponse{
		Message: "Payment successfully processed. Menu purchased.",
		Success: true,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /menu/event/:placeId/:menuId/payment-session
func CreateMenuPaymentSession(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		menuId := ps.ByName("menuid")
		placeId := ps.ByName("placeid")

		// Parse request body for stock
		var body struct {
			Stock int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Stock < 1 {
			http.Error(w, "Invalid request or stock", http.StatusBadRequest)
			return
		}

		// Generate a Stripe payment session
		session, err := stripe.CreateMenuSession(menuId, placeId, body.Stock)
		if err != nil {
			log.Printf("Error creating payment session: %v", err)
			http.Error(w, "Failed to create payment session", http.StatusInternalServerError)
			return
		}

		// Respond with the session URL
		dataResponse := map[string]any{
			"paymentUrl": session.URL,
			"placeid":    session.PlaceID,
			"menuid":     session.MenuID,
			"stock":      session.Stock,
		}

		// Respond with the session URL
		response := map[string]any{
			"success": true,
			"data":    dataResponse,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
