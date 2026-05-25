package itinerary

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

// normalizeItinerary ensures slices are never nil
func normalizeItinerary(it *models.Itinerary) {
	if it.Days == nil {
		it.Days = []models.Day{}
		return
	}
	for i := range it.Days {
		if it.Days[i].Visits == nil {
			it.Days[i].Visits = []models.Visit{}
		}
	}
}

// extract user ID from request
func GetRequestingUserID(_ http.ResponseWriter, r *http.Request) (string, error) {
	return utils.GetUserIDFromRequest(r), nil
}

// POST /api/itineraries
func CreateItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var it models.Itinerary
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		userID, err := GetRequestingUserID(w, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		it.ItineraryID = utils.GenerateRandomString(13)
		it.UserID = userID
		it.Published = false
		if it.Status == "" {
			it.Status = "Draft"
		}

		normalizeItinerary(&it)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.Insert(ctx, ItineraryCollection, it); err != nil {
			http.Error(w, "Error inserting itinerary", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, it)
	}
}

// PUT /api/itineraries/:id
func UpdateItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID, err := GetRequestingUserID(w, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		itineraryID := ps.ByName("id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var existing models.Itinerary
		if err := app.DB.FindOne(ctx, ItineraryCollection, map[string]any{
			"itineraryid": itineraryID,
			"deleted":     map[string]any{"$ne": true},
		}, &existing); err != nil {
			http.Error(w, "Itinerary not found", http.StatusNotFound)
			return
		}

		if existing.UserID != userID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var updated models.Itinerary
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		normalizeItinerary(&updated)

		update := map[string]any{
			"$set": map[string]any{
				"name":        updated.Name,
				"description": updated.Description,
				"start_date":  updated.StartDate,
				"end_date":    updated.EndDate,
				"status":      updated.Status,
				"published":   updated.Published,
				"days":        updated.Days,
			},
		}

		if err := app.DB.UpdateOne(ctx, ItineraryCollection, map[string]any{"itineraryid": itineraryID}, update); err != nil {
			http.Error(w, "Error updating itinerary", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"message": "Itinerary updated successfully"})
	}
}

// DELETE /api/itineraries/:id (soft delete)
func DeleteItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID, err := GetRequestingUserID(w, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		itineraryID := ps.ByName("id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		update := map[string]any{"$set": map[string]any{"deleted": true}}
		if err := app.DB.UpdateOne(ctx, ItineraryCollection, map[string]any{"itineraryid": itineraryID, "userid": userID}, update); err != nil {
			http.Error(w, "Itinerary not found or forbidden", http.StatusNotFound)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"message": "Itinerary deleted"})
	}
}

// POST /api/itineraries/:id/fork
func ForkItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID, err := GetRequestingUserID(w, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		originalID := ps.ByName("id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var original models.Itinerary
		if err := app.DB.FindOne(ctx, ItineraryCollection, map[string]any{
			"itineraryid": originalID,
			"deleted":     map[string]any{"$ne": true},
		}, &original); err != nil {
			http.Error(w, "Original itinerary not found", http.StatusNotFound)
			return
		}

		normalizeItinerary(&original)

		newItinerary := models.Itinerary{
			ItineraryID: utils.GenerateRandomString(13),
			UserID:      userID,
			Name:        "Forked - " + original.Name,
			Description: original.Description,
			StartDate:   original.StartDate,
			EndDate:     original.EndDate,
			Days:        original.Days,
			Status:      "Draft",
			Published:   false,
			ForkedFrom:  &originalID,
		}

		if err := app.DB.Insert(ctx, ItineraryCollection, newItinerary); err != nil {
			http.Error(w, "Error forking itinerary", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, newItinerary)
	}
}

// PUT /api/itineraries/:id/publish
func PublishItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID, err := GetRequestingUserID(w, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := ps.ByName("id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		update := map[string]any{"$set": map[string]any{"published": true}}
		if err := app.DB.UpdateOne(ctx, ItineraryCollection, map[string]any{"itineraryid": id, "userid": userID}, update); err != nil {
			http.Error(w, "Itinerary not found or forbidden", http.StatusNotFound)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"published": true})
	}
}
