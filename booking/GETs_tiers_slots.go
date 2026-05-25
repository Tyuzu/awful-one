package booking

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
)

func ListSlots(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		entityType := r.URL.Query().Get("entityType")
		entityID := r.URL.Query().Get("entityId")

		filter := bson.M{}
		if entityType != "" {
			filter["entityType"] = entityType
		}
		if entityID != "" {
			filter["entityId"] = entityID
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var slots []Slot
		if err := app.DB.FindMany(ctx, slotsCollection, filter, &slots); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"slots": slots,
		})
	}
}

func ListBookings(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		entityType := r.URL.Query().Get("entityType")
		entityID := r.URL.Query().Get("entityId")
		status := r.URL.Query().Get("status")

		filter := bson.M{}
		if entityType != "" {
			filter["entityType"] = entityType
		}
		if entityID != "" {
			filter["entityId"] = entityID
		}
		if status != "" {
			filter["status"] = status
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var bookings []Booking
		if err := app.DB.FindMany(ctx, bookingsCollection, filter, &bookings); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"bookings": bookings,
		})
	}
}

func GetDateCapacity(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		entityType := r.URL.Query().Get("entityType")
		entityID := r.URL.Query().Get("entityId")
		date := r.URL.Query().Get("date")

		if entityType == "" || entityID == "" || date == "" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var dc DateCap
		err := app.DB.FindOne(
			ctx,
			dateCapsCollection,
			bson.M{
				"entityType": entityType,
				"entityId":   entityID,
				"date":       date,
			},
			&dc,
		)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"capacity": nil,
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"capacity": dc.Capacity,
		})
	}
}

func ListTiers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		entityType := r.URL.Query().Get("entityType")
		entityID := r.URL.Query().Get("entityId")

		filter := bson.M{}
		if entityType != "" {
			filter["entityType"] = entityType
		}
		if entityID != "" {
			filter["entityId"] = entityID
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var tiers []Tier
		if err := app.DB.FindMany(ctx, tiersCollection, filter, &tiers); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"tiers": tiers,
		})
	}
}
