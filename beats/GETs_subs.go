package beats

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"
)

// GET /api/v1/subscribes/:type/:id
func DoesSubscribeEntity(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		currentUserID := utils.GetUserIDFromRequest(r)
		if currentUserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		entityID := ps.ByName("id")
		if entityID == "" {
			http.Error(w, "Entity ID required", http.StatusBadRequest)
			return
		}

		entityType := ps.ByName("type")

		switch entityType {
		case "user", "artist", "feedpost":
			// allowed
		default:
			http.Error(w, "Invalid entity type", http.StatusBadRequest)
			return
		}

		count, err := app.DB.CountDocuments(
			r.Context(),
			subscribersCollection,
			bson.M{
				"userid": currentUserID,
				"subscribed": bson.M{
					"$in": []string{entityID},
				},
			},
		)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp := map[string]bool{
			"hasSubscribed": count > 0,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// GET /api/v1/subscribers/:id
func GetSubscribers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		targetUserID := ps.ByName("id")
		if targetUserID == "" {
			http.Error(w, "Target user ID required", http.StatusBadRequest)
			return
		}

		var sub models.UserSubscribe
		err := app.DB.FindOne(
			r.Context(),
			subscribersCollection,
			bson.M{"userid": targetUserID},
			&sub,
		)
		if err != nil || len(sub.Subscribers) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]models.User{})
			return
		}

		var subscribers []models.User
		err = app.DB.FindMany(
			r.Context(),
			usersCollection,
			bson.M{
				"userid": bson.M{
					"$in": sub.Subscribers,
				},
			},
			&subscribers,
		)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(subscribers)
	}
}
