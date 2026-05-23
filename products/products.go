package products

import (
	"encoding/json"
	"naevis/infra"
	"naevis/models"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func GetProductDetails(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		entityType := ps.ByName("entityType")
		entityId := ps.ByName("entityId")

		var product models.Product

		switch entityType {
		case "product":
			product = getProductEntity(ctx, entityId, app)
		case "tool":
			product = getToolEntity(entityId)
		case "subscription":
			product = getSubscriptionEntity(entityId)
		case "media":
			product = getMediaEntity(entityId)
		case "fmcg":
			product = getFMCGEntity(entityId)
		case "art":
			product = getArtEntity(entityId)
		default:
			http.Error(w, "Invalid entity type", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(product)
	}
}

// --- entity handlers ---

func getToolEntity(id string) models.Product {
	return models.Product{
		ProductID:   id,
		Name:        "Hand Tool",
		Description: "Durable metal tool",
		Price:       49.99,
		Unit:        "unit",
		Specs: map[string]string{
			"Material": "Alloy",
			"Use":      "Mechanical work",
		},
		Images: []string{
			"8d8b89cd-8358-4006-96fc-5353305941d7",
			"example2.jpg",
		},
	}
}

func getSubscriptionEntity(id string) models.Product {
	return models.Product{
		ProductID:    id,
		Name:         "Premium Subscription",
		Description:  "Access to all features",
		Price:        19.99,
		BillingCycle: "Monthly",
		TrialPeriod:  "14 days",
		Scope:        "All services",
		Duration:     "1 month",
		Images: []string{
			"8d8b89cd-8358-4006-96fc-5353305941d7",
			"example2.jpg",
		},
	}
}

func getMediaEntity(id string) models.Product {
	return models.Product{
		ProductID:   id,
		Name:        "E-book",
		Description: "Digital reading material",
		Author:      "John Doe",
		ISBN:        "123-4567890123",
		Platform:    "Kindle",
		Version:     "2.0",
		License:     "Single-user",
		Price:       9.99,
		Images: []string{
			"8d8b89cd-8358-4006-96fc-5353305941d7",
			"example2.jpg",
		},
	}
}

func getFMCGEntity(id string) models.Product {
	return models.Product{
		ProductID:   id,
		Name:        "Snack Pack",
		Description: "Ready-to-eat food item",
		Price:       2.49,
		Unit:        "pack",
		Ingredients: "Potato, Salt, Oil",
		ExpiryDate:  "2025-12-31",
		Weight:      "100g",
		Images: []string{
			"8d8b89cd-8358-4006-96fc-5353305941d7",
			"example2.jpg",
		},
	}
}

func getArtEntity(id string) models.Product {
	return models.Product{
		ProductID:   id,
		Name:        "Abstract Painting",
		Description: "Original hand-painted artwork",
		Artist:      "Jane Artist",
		Medium:      "Oil on Canvas",
		Dimensions:  "24x36 inches",
		Price:       499.99,
		Images: []string{
			"8d8b89cd-8358-4006-96fc-5353305941d7",
			"example2.jpg",
		},
	}
}
