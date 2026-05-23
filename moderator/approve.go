package moderator

import (
	"encoding/json"
	"net/http"
	"time"

	"naevis/infra"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// ---------------------- List Moderator Applications ----------------------
// Optional query param: ?status=pending|approved|rejected
func ListModeratorApplications(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		status := r.URL.Query().Get("status")

		filter := bson.M{}
		if status != "" {
			filter["status"] = status
		}

		var applications []ModeratorApplication
		err := app.DB.FindMany(
			ctx,
			moderatorAppsCollection,
			filter,
			&applications,
		)
		if err != nil {
			http.Error(w, `{"error":"Failed to fetch applications"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(applications)
	}
}

// ---------------------- Approve Moderator Application ----------------------
func ApproveModerator(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		id := ps.ByName("id")
		if id == "" {
			http.Error(w, `{"error":"Invalid ID"}`, http.StatusBadRequest)
			return
		}

		err := app.DB.UpdateOne(
			ctx,
			moderatorAppsCollection,
			bson.M{"id": id},
			bson.M{
				"$set": bson.M{
					"status":    "approved",
					"updatedAt": time.Now().UTC(),
				},
			},
		)
		if err != nil {
			http.Error(w, `{"error":"Application not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Application approved",
		})
	}
}

// ---------------------- Reject Moderator Application ----------------------
func RejectModerator(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		id := ps.ByName("id")
		if id == "" {
			http.Error(w, `{"error":"Invalid ID"}`, http.StatusBadRequest)
			return
		}

		err := app.DB.UpdateOne(
			ctx,
			moderatorAppsCollection,
			bson.M{"id": id},
			bson.M{
				"$set": bson.M{
					"status":    "rejected",
					"updatedAt": time.Now().UTC(),
				},
			},
		)
		if err != nil {
			http.Error(w, `{"error":"Application not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Application rejected",
		})
	}
}
