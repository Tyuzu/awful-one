package farms

import (
	"context"
	"encoding/json"
	"fmt"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --------------------------------------------------
// Create
// --------------------------------------------------

func CreateProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		createItem(w, r, "product", app)
	}
}

func CreateTool(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		createItem(w, r, "tool", app)
	}
}

func createItem(w http.ResponseWriter, r *http.Request, itemType string, app *infra.Deps) {
	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	item, err := parseProductJSON(r, itemType)
	if err != nil {
		http.Error(w, "Failed to parse body: "+err.Error(), http.StatusBadRequest)
		return
	}

	item.ProductID = utils.GenerateRandomString(13)
	item.UserID = userID
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := app.DB.InsertOne(ctx, productsCollection, item); err != nil {
		http.Error(w, "Failed to insert item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(item)
}

// --------------------------------------------------
// Update
// --------------------------------------------------

func UpdateProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		updateItem(w, r, ps, "product", app)
	}
}

func UpdateTool(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		updateItem(w, r, ps, "tool", app)
	}
}

func updateItem(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	itemType string,
	app *infra.Deps,
) {
	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := ps.ByName("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// SECURITY: Check if user is the creator
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var existingItem models.Product
	if err := app.DB.FindOne(ctx, productsCollection, bson.M{"productid": id}, &existingItem); err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	if existingItem.UserID != userID {
		http.Error(w, "Forbidden: Only creator can update this product", http.StatusForbidden)
		return
	}

	item, err := parseProductJSON(r, itemType)
	if err != nil {
		http.Error(w, "Failed to parse body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Preserve original UserID and CreatedAt
	item.UserID = existingItem.UserID
	item.CreatedAt = existingItem.CreatedAt
	item.UpdatedAt = time.Now()

	update := bson.M{"$set": item}
	if err := app.DB.UpdateOne(ctx, productsCollection, bson.M{"productid": id}, update); err != nil {
		http.Error(w, "Failed to update item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(utils.M{"status": "updated"})
}

// --------------------------------------------------
// Delete
// --------------------------------------------------

func DeleteProduct(app *infra.Deps) httprouter.Handle {
	return deleteItem(app)
}

func DeleteTool(app *infra.Deps) httprouter.Handle {
	return deleteItem(app)
}

func deleteItem(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := ps.ByName("id")
		if id == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// SECURITY: Check if user is the creator
		var item models.Product
		if err := app.DB.FindOne(ctx, productsCollection, bson.M{"productid": id}, &item); err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}

		if item.UserID != userID {
			http.Error(w, "Forbidden: Only creator can delete this product", http.StatusForbidden)
			return
		}

		if _, err := app.DB.DeleteOne(ctx, productsCollection, bson.M{"productid": id}); err != nil {
			http.Error(w, "Failed to delete item", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(utils.M{"status": "deleted"})
	}
}

// --------------------------------------------------
// Parsing
// --------------------------------------------------

// parseProductJSON parses a JSON body into models.Product
func parseProductJSON(r *http.Request, itemType string) (models.Product, error) {
	var payload struct {
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		Category      string   `json:"category"`
		SKU           string   `json:"sku"`
		Unit          string   `json:"unit"`
		Featured      bool     `json:"featured"`
		Price         float64  `json:"price"`
		Quantity      float64  `json:"quantity"`
		AvailableFrom string   `json:"availableFrom"`
		AvailableTo   string   `json:"availableTo"`
		Images        []string `json:"images"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return models.Product{}, fmt.Errorf("invalid JSON: %w", err)
	}

	item := models.Product{
		Name:        payload.Name,
		Description: payload.Description,
		Category:    payload.Category,
		SKU:         payload.SKU,
		Unit:        payload.Unit,
		Type:        itemType,
		Featured:    payload.Featured,
		Price:       payload.Price,
		Quantity:    payload.Quantity,
		Images:      payload.Images,
	}

	if payload.AvailableFrom != "" {
		if t, err := time.Parse("2006-01-02", payload.AvailableFrom); err == nil {
			item.AvailableFrom = &models.SafeTime{Time: t}
		}
	}
	if payload.AvailableTo != "" {
		if t, err := time.Parse("2006-01-02", payload.AvailableTo); err == nil {
			item.AvailableTo = &models.SafeTime{Time: t}
		}
	}

	return item, nil
}
