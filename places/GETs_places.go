package places

import (
	"context"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"

	"go.mongodb.org/mongo-driver/bson"
)

// --- Get all places (summary) ---
func GetPlaces(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var places []models.Place
		if err := app.DB.FindMany(ctx, placesCollection, bson.M{}, &places); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch places")
			return
		}

		var result []models.PlacesResponse
		for _, p := range places {
			desc := p.Description
			if len(desc) > 60 {
				desc = desc[:60] + "..."
			}

			tags := p.Tags
			if len(tags) > 5 {
				tags = tags[:5]
			}

			result = append(result, models.PlacesResponse{
				PlaceID:        p.PlaceID,
				Name:           p.Name,
				ShortDesc:      desc,
				Address:        p.Address,
				Distance:       p.Distance,
				OperatingHours: p.OperatingHours,
				Category:       p.Category,
				Tags:           tags,
				Banner:         p.Banner,
			})
		}

		utils.RespondWithJSON(w, http.StatusOK, result)
	}
}

// --- Get single place by path param ---
func GetPlace(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		placeID := strings.TrimSpace(ps.ByName("placeid"))
		if placeID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Place ID is required")
			return
		}

		var place models.Place
		if err := app.DB.FindOne(
			r.Context(),
			placesCollection,
			bson.M{"placeid": placeID},
			&place,
		); err != nil {

			utils.RespondWithError(w, http.StatusNotFound, "Place not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, place)
	}
}

// --- Get single place by query param ---
func GetPlaceQ(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		placeID := strings.TrimSpace(r.URL.Query().Get("id"))
		if placeID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Place ID is required")
			return
		}

		var place models.Place
		if err := app.DB.FindOne(
			r.Context(),
			placesCollection,
			bson.M{"placeid": placeID},
			&place,
		); err != nil {

			utils.RespondWithError(w, http.StatusNotFound, "Place not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, place)
	}
}
