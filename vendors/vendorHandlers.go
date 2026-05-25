package vendors

import (
	"context"
	"encoding/json"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// RegisterVendorHandler handles vendor registration
func RegisterVendorHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Extract user ID from auth context (middleware should have set this)
		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Name        string `json:"name"`
			Category    string `json:"category"`
			Description string `json:"description"`
			Phone       string `json:"phone"`
			Email       string `json:"email"`
			Location    string `json:"location"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Category == "" {
			http.Error(w, "Name and category are required", http.StatusBadRequest)
			return
		}

		vendor, err := RegisterVendor(ctx, app, userID, req.Name, req.Category, req.Description)
		if err != nil {
			http.Error(w, "Failed to register vendor: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(vendor)
	}
}

// GetVendorsHandler retrieves all available vendors
func GetVendorsHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		category := r.URL.Query().Get("category")

		var vendors []models.Vendor
		var err error

		if category != "" {
			vendors, err = GetVendorsByCategory(ctx, app, category)
		} else {
			vendors, err = GetAllVendors(ctx, app)
		}

		if err != nil {
			http.Error(w, "Failed to get vendors: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if vendors == nil {
			vendors = []models.Vendor{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vendors)
	}
}

// GetVendorHandler retrieves a single vendor by ID
func GetVendorHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		vendorID := ps.ByName("vendorID")
		vendor, err := GetVendorByID(ctx, app, vendorID)
		if err != nil {
			http.Error(w, "Vendor not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vendor)
	}
}

// UpdateVendorHandler updates vendor information
func UpdateVendorHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vendorID := ps.ByName("vendorID")

		// Verify vendor belongs to user
		vendor, err := GetVendorByID(ctx, app, vendorID)
		if err != nil {
			http.Error(w, "Vendor not found", http.StatusNotFound)
			return
		}

		if vendor.UserID != userID {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		err = UpdateVendor(ctx, app, vendorID, nil)
		if err != nil {
			http.Error(w, "Failed to update vendor", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Vendor updated successfully"})
	}
}

// HireVendorHandler handles hiring a vendor for an event
func HireVendorHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		eventID := ps.ByName("eventID")

		var req struct {
			VendorID string `json:"vendorid"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Get vendor info
		vendor, err := GetVendorByID(ctx, app, req.VendorID)
		if err != nil {
			http.Error(w, "Vendor not found", http.StatusNotFound)
			return
		}

		// Hire vendor
		hiring, err := HireVendor(ctx, app, eventID, req.VendorID, vendor.Name, vendor.Category, userID)
		if err != nil {
			if err.Error() == "vendor already hired for this event" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "ALREADY_HIRED"})
				return
			}
			http.Error(w, "Failed to hire vendor: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(hiring)
	}
}

// GetEventVendorsHandler retrieves all vendors hired for an event
func GetEventVendorsHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		eventID := ps.ByName("eventID")

		vendorResponses, err := GetVendorsByEvent(ctx, app, eventID)
		if err != nil {
			http.Error(w, "Failed to get vendors: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if vendorResponses == nil {
			vendorResponses = []models.VendorResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vendorResponses)
	}
}

// RemoveVendorHandler removes a vendor from an event
func RemoveVendorHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		eventID := ps.ByName("eventID")
		vendorID := ps.ByName("vendorID")

		err := RemoveVendorFromEvent(ctx, app, eventID, vendorID)
		if err != nil {
			http.Error(w, "Failed to remove vendor: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Vendor removed successfully"})
	}
}
