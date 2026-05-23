package places

import (
	"net/http"
	"strings"
	"time"

	"naevis/dels"
	"naevis/globals"
	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"

	"go.mongodb.org/mongo-driver/bson"
)

// --- EditPlace endpoint ---
func EditPlace(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		placeID := strings.TrimSpace(ps.ByName("placeid"))
		if placeID == "" {
			http.Error(w, "Place ID is required", http.StatusBadRequest)
			return
		}

		requestingUserID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Invalid user", http.StatusUnauthorized)
			return
		}

		// Fetch existing place (use placeid)
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

		if existing.CreatedBy != requestingUserID {
			http.Error(w, "You are not authorized to edit this place", http.StatusForbidden)
			return
		}

		// Parse update fields
		_, updateFields, err := parseAndBuildPlace(r, "update")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(updateFields) == 0 {
			http.Error(w, "No fields to update", http.StatusBadRequest)
			return
		}

		updateFields["updated_at"] = time.Now()
		updateFields["updatedBy"] = requestingUserID

		// ✅ Update using placeid and plain fields
		if err := app.DB.Update(
			ctx,
			placesCollection,
			bson.M{"placeid": placeID},
			updateFields,
		); err != nil {
			http.Error(w, "Failed to update place", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, updateFields)
	}
}

// --- DeletePlace endpoint ---
func DeletePlace(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Delegate to your dels.DeletePlace logic, which should handle DB + cache
		dels.DeletePlace(app)
	}
}
