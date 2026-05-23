package places

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// 🏟️ Events

func GetEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("GetEvent not implemented yet"))
	}
}

func PostEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("PostEvent not implemented yet"))
	}
}

func PutEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("PutEvent not implemented yet"))
	}
}

func DeleteEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("DeleteEvent not implemented yet"))
	}
}

func PostViewEventDetails(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("PostViewEventDetails not implemented yet"))
	}
}

func GetEvents(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")

		placeID := ps.ByName("placeid")
		if placeID == "" {
			http.Error(w, "missing required path parameter: placeid", http.StatusBadRequest)
			return
		}

		// Pagination
		page := 1
		limit := 10

		if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}

		skip := (page - 1) * limit
		now := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		// Filter: upcoming events for this place
		filter := map[string]any{
			"placeid":  placeID,
			"date_gte": now,
		}

		// Count total
		total, err := app.DB.Count(ctx, eventsCollection, filter)
		if err != nil {
			http.Error(w, "Failed to count events", http.StatusInternalServerError)
			return
		}

		// Fetch events
		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort:  bson.D{{Key: "date", Value: 1}},
			Projection: []string{
				"eventid",
				"title",
				"description",
				"start_date_time",
				"end_date_time",
				"placename",
				"banner_image",
				"category",
			},
		}

		var events []models.Event
		if err := app.DB.FindManyWithOptions(ctx, eventsCollection, filter, opts, &events); err != nil {
			http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
			return
		}

		if events == nil {
			events = []models.Event{}
		}

		response := map[string]any{
			"events": events,
			"total":  total,
			"page":   page,
			"limit":  limit,
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}
