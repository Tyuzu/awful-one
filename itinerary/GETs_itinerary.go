package itinerary

import (
	"context"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// GET /api/itineraries/all/:id
func GetItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		itineraryID := ps.ByName("id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var itinerary models.Itinerary
		if err := app.DB.FindOne(ctx, ItineraryCollection, map[string]any{
			"itineraryid": itineraryID,
			"deleted":     map[string]any{"$ne": true},
		}, &itinerary); err != nil {
			http.Error(w, "Itinerary not found", http.StatusNotFound)
			return
		}

		normalizeItinerary(&itinerary)
		utils.RespondWithJSON(w, http.StatusOK, itinerary)
	}
}

// GET /api/itineraries
func GetItineraries(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var itineraries []models.Itinerary
		if err := app.DB.FindMany(ctx, ItineraryCollection, map[string]any{"deleted": map[string]any{"$ne": true}}, &itineraries); err != nil {
			http.Error(w, "Error fetching itineraries", http.StatusInternalServerError)
			return
		}

		if itineraries == nil {
			itineraries = []models.Itinerary{}
		}

		for i := range itineraries {
			normalizeItinerary(&itineraries[i])
		}

		utils.RespondWithJSON(w, http.StatusOK, itineraries)
	}
}

// GET /api/itineraries/search
func SearchItineraries(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		query := r.URL.Query()

		filter := map[string]any{"deleted": map[string]any{"$ne": true}}
		if start := query.Get("start_date"); start != "" {
			filter["start_date"] = start
		}
		if location := query.Get("location"); location != "" {
			filter["days.visits.location"] = map[string]any{"$in": []string{location}}
		}
		if status := query.Get("status"); status != "" {
			filter["status"] = status
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var itineraries []models.Itinerary
		if err := app.DB.FindMany(ctx, ItineraryCollection, filter, &itineraries); err != nil {
			http.Error(w, "Error fetching itineraries", http.StatusInternalServerError)
			return
		}

		if itineraries == nil {
			itineraries = []models.Itinerary{}
		}

		for i := range itineraries {
			normalizeItinerary(&itineraries[i])
		}

		utils.RespondWithJSON(w, http.StatusOK, itineraries)
	}
}
