package suggestions

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func SuggestFollowers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		currentUserID := r.URL.Query().Get("userid")
		if currentUserID == "" {
			http.Error(w, "Missing userid", http.StatusBadRequest)
			return
		}

		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit < 1 {
			limit = 10
		}

		offset := int64((page - 1) * limit)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var followData models.UserFollow
		err = app.DB.FindOne(
			ctx,
			followingsCollection,
			bson.M{"userid": currentUserID},
			&followData,
		)
		if err != nil {
			followData.Follows = []string{}
		}

		excludedUserIDs := append(followData.Follows, currentUserID, userID)

		filter := bson.M{
			"userid": bson.M{"$nin": excludedUserIDs},
		}

		var users []models.UserSuggest
		if err := app.DB.FindMany(
			ctx,
			usersCollection,
			filter,
			&users,
		); err != nil {
			http.Error(w, "Failed to fetch suggestions", http.StatusInternalServerError)
			return
		}

		for i := range users {
			users[i].IsFollowing = false
		}

		utils.SortAndSlice(
			&users,
			bson.D{{Key: "userid", Value: 1}},
			offset,
			int64(limit),
		)

		if users == nil {
			users = []models.UserSuggest{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(users)
	}
}

func GetNearbyPlaces(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		curplace := r.URL.Query().Get("place")
		if len(curplace) != 14 {
			log.Printf("invalid place id: %s", curplace)
		}

		var places []models.Place
		if err := app.DB.FindMany(
			ctx,
			placesCollection,
			bson.M{},
			&places,
		); err != nil {
			http.Error(w, "Failed to fetch places", http.StatusInternalServerError)
			return
		}

		if places == nil {
			places = []models.Place{}
		}

		sanitized := make([]map[string]any, 0, len(places))
		for _, place := range places {
			if place.PlaceID == curplace {
				continue
			}
			sanitized = append(sanitized, map[string]any{
				"placeid":     place.PlaceID,
				"name":        place.Name,
				"banner":      place.Banner,
				"category":    place.Category,
				"capacity":    place.Capacity,
				"reviewCount": place.ReviewCount,
			})
		}

		_ = json.NewEncoder(w).Encode(sanitized)
	}
}
