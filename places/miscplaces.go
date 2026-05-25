package places

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// UpdatePlaceInfo updates accessibility and amenities for a place
func UpdatePlaceInfo(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		placeID := strings.TrimSpace(ps.ByName("placeid"))
		if placeID == "" {
			http.Error(w, "Place ID is required", http.StatusBadRequest)
			return
		}

		userID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Invalid user", http.StatusUnauthorized)
			return
		}

		// Fetch existing place (use placeid, NOT _id)
		var existing struct {
			CreatedBy string `bson:"createdBy"`
		}
		if err := app.DB.FindOne(
			ctx,
			placesCollection,
			bson.M{"placeid": placeID},
			&existing,
		); err != nil {
			http.Error(w, "Place not found", http.StatusNotFound)
			return
		}

		if existing.CreatedBy != userID {
			http.Error(w, "Not authorized", http.StatusForbidden)
			return
		}

		// Parse JSON payload
		var payload struct {
			AccessibilityInfo string   `json:"accessibility_info"`
			Amenities         []string `json:"amenities"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		update := bson.M{}

		if payload.AccessibilityInfo != "" {
			update["accessibility_info"] = payload.AccessibilityInfo
		}

		if len(payload.Amenities) > 0 {
			set := map[string]struct{}{}
			for _, a := range payload.Amenities {
				a = strings.TrimSpace(a)
				if a != "" {
					set[a] = struct{}{}
				}
			}

			if len(set) > 0 {
				amenities := make([]string, 0, len(set))
				for a := range set {
					amenities = append(amenities, a)
				}
				update["amenities"] = amenities
			}
		}

		if len(update) == 0 {
			http.Error(w, "No fields to update", http.StatusBadRequest)
			return
		}

		update["updated_at"] = time.Now()
		update["updatedBy"] = userID

		// ✅ Pass plain fields (DB layer adds $set)
		if err := app.DB.Update(
			ctx,
			placesCollection,
			bson.M{"placeid": placeID},
			update,
		); err != nil {
			http.Error(w, "Failed to update place info", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, update)
	}
}
