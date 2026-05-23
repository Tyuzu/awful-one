package events

import (
	"context"
	"encoding/json"
	"log"
	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// GetEvent fetches a single event with its tickets, media, and merch
func GetEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		eventID := ps.ByName("eventid")
		if eventID == "" {
			http.Error(w, "Missing event ID", http.StatusBadRequest)
			return
		}

		// Aggregate pipeline (simplified as map slices)
		pipeline := []any{
			map[string]any{"$match": map[string]any{"eventid": eventID}},
			map[string]any{"$lookup": map[string]any{
				"from":         "ticks",
				"localField":   "eventid",
				"foreignField": "eventid",
				"as":           "tickets",
			}},
			map[string]any{"$lookup": map[string]any{
				"from": "media",
				"let":  map[string]any{"eid": "$eventid"},
				"pipeline": []any{
					map[string]any{"$match": map[string]any{
						"$expr": map[string]any{
							"$and": []any{
								map[string]any{"$eq": []any{"$entityid", "$$eid"}},
								map[string]any{"$eq": []any{"$entitytype", "event"}},
							},
						},
					}},
				},
				"as": "media",
			}},
			map[string]any{"$lookup": map[string]any{
				"from": "merch",
				"let":  map[string]any{"eid": "$eventid"},
				"pipeline": []any{
					map[string]any{"$match": map[string]any{
						"$expr": map[string]any{
							"$and": []any{
								map[string]any{"$eq": []any{"$entity_id", "$$eid"}},
								map[string]any{"$eq": []any{"$entity_type", "event"}},
							},
						},
					}},
				},
				"as": "merch",
			}},
		}

		var events []models.Event
		if err := app.DB.Aggregate(ctx, eventsCollection, pipeline, &events); err != nil {
			log.Println("Aggregate error:", err)
			http.Error(w, "Failed to fetch event", http.StatusInternalServerError)
			return
		}

		if len(events) == 0 {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}

		safe := toSafeEvent(events[0])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(safe)
	}
}

// GetEvents fetches paginated events
func GetEvents(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		skip, limit := utils.ParsePagination(r, 10, 100)
		filter := map[string]any{} // optionally {"published": true}

		totalCount, err := app.DB.CountDocuments(ctx, eventsCollection, filter)
		if err != nil {
			log.Println("CountDocuments error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch event count")
			return
		}

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var rawEvents []models.Event
		if err := app.DB.FindManyWithOptions(ctx, eventsCollection, filter, opts, &rawEvents); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch events")
			return
		}

		safeEvents := make([]models.Event, 0, len(rawEvents))
		for _, e := range rawEvents {
			safeEvents = append(safeEvents, toSafeEvent(e))
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"events":     safeEvents,
			"eventCount": totalCount,
			"page":       skip/limit + 1,
			"limit":      limit,
		})
	}
}

// GetEventsCount returns the total count of published events.
func GetEventsCount(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Example static count; replace with a real DB query if needed.
		count := 3
		utils.RespondWithJSON(w, http.StatusOK, count)
	}
}
