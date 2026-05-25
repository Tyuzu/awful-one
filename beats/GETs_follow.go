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

// GET /api/v1/follow/:id
func DoesFollow(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		followedUserID := ps.ByName("id")

		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if followedUserID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		count, err := app.DB.CountDocuments(
			r.Context(),
			followingsCollection,
			bson.M{
				"userid": userID,
				"follows": bson.M{
					"$in": []string{followedUserID},
				},
			},
		)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := map[string]bool{
			"isFollowing": count > 0,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}

// GET /api/v1/followers
func GetFollowers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var userFollow models.UserFollow
		err := app.DB.FindOne(
			r.Context(),
			followingsCollection,
			bson.M{"userid": userID},
			&userFollow,
		)
		if err != nil || len(userFollow.Followers) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]models.User{})
			return
		}

		var followers []models.User
		err = app.DB.FindMany(
			r.Context(),
			usersCollection,
			bson.M{
				"userid": bson.M{
					"$in": userFollow.Followers,
				},
			},
			&followers,
		)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(followers)
	}
}

// GET /api/v1/following
func GetFollowing(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var userFollow models.UserFollow
		err := app.DB.FindOne(
			r.Context(),
			followingsCollection,
			bson.M{"userid": userID},
			&userFollow,
		)
		if err != nil || len(userFollow.Follows) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]models.User{})
			return
		}

		var following []models.User
		err = app.DB.FindMany(
			r.Context(),
			usersCollection,
			bson.M{
				"userid": bson.M{
					"$in": userFollow.Follows,
				},
			},
			&following,
		)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(following)
	}
}
