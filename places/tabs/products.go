package places

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"naevis/infra"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product represents a product sold by a place
type Product struct {
	ID      primitive.ObjectID `json:"_id"`
	PlaceID string             `json:"placeid"`
	Name    string             `json:"name"`
	Price   float64            `json:"price"`
}

// UnmarshalJSON supports both string and float for price
func (p *Product) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	name, ok := raw["name"].(string)
	if !ok || name == "" {
		return errors.New("name is required and must be a string")
	}
	p.Name = name

	if placeID, ok := raw["placeid"].(string); ok {
		p.PlaceID = placeID
	}

	switch price := raw["price"].(type) {
	case float64:
		p.Price = price
	case string:
		var parsed float64
		if err := json.Unmarshal([]byte(price), &parsed); err != nil {
			return errors.New("price must be a number")
		}
		p.Price = parsed
	default:
		return errors.New("price must be a number or numeric string")
	}

	return nil
}

// validateProduct ensures required fields are valid
func validateProduct(p Product) error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.Price <= 0 {
		return errors.New("price must be positive")
	}
	return nil
}

// parseJSON checks content-type, reads and parses JSON body
func parseJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct != "application/json" {
		http.Error(w, "Expected application/json", http.StatusUnsupportedMediaType)
		return errors.New("unsupported content type")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, dst); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}

func GetProducts(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		placeID := ps.ByName("placeid")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var products []Product
		if err := app.DB.FindMany(ctx,
			productsCollection,
			map[string]any{"placeid": placeID},
			&products,
		); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if products == nil {
			products = make([]Product, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(products)
	}
}

func PostProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		placeID := ps.ByName("placeid")

		var product Product
		if err := parseJSON(w, r, &product); err != nil {
			return
		}

		product.ID = primitive.NewObjectID()
		product.PlaceID = placeID

		if err := validateProduct(product); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.InsertOne(ctx, productsCollection, product); err != nil {
			http.Error(w, "Insert failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(product)
	}
}

func PutProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := primitive.ObjectIDFromHex(ps.ByName("productId"))
		if err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		var updateData Product
		if err := parseJSON(w, r, &updateData); err != nil {
			return
		}

		if err := validateProduct(updateData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.UpdateOne(ctx,
			productsCollection,
			map[string]any{"_id": id},
			map[string]any{
				"name":  updateData.Name,
				"price": updateData.Price,
			},
		); err != nil {
			http.Error(w, "Update failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func DeleteProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := primitive.ObjectIDFromHex(ps.ByName("productId"))
		if err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if _, err := app.DB.DeleteOne(ctx,
			productsCollection,
			map[string]any{"_id": id},
		); err != nil {
			http.Error(w, "Delete failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func PostPlaceProductPurchase(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if _, err := primitive.ObjectIDFromHex(ps.ByName("productId")); err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}

		// NOTE: IMPORTANT - Implement audit logging for purchases
		// Should log:
		// - Purchase timestamp, user ID, product ID, quantity
		// - Inventory mutation event to message queue
		// - Ensure transaction integrity before accepting payment

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// Optional fallbacks
func GetProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

func PostProductPurchase(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}
