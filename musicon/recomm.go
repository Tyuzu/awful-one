package musicon

import (
	"context"
	"naevis/infra"
	"naevis/infra/db"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --------------------------- Helpers ---------------------------

func sanitizePagination(limit, page int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if page <= 0 {
		page = 1
	}
	return limit, page
}

// --------------------------- Recommendations ---------------------------

func GetRecommendedSongs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		limit, page := getPaginationParams(r)
		limit, page = sanitizePagination(limit, page)

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  (page - 1) * limit,
			Sort:  bson.D{{Key: "plays", Value: -1}, {Key: "_id", Value: -1}},
		}

		filter := bson.M{"published": true}

		songs := []Song{}
		if err := app.DB.FindManyWithOptions(ctx, songsCollection, filter, opts, &songs); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch recommended songs")
			return
		}

		respondJSON(w, http.StatusOK, songs, "Recommended songs fetched")
	}
}

func GetRecommendedAlbums(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		limit, page := getPaginationParams(r)
		limit, page = sanitizePagination(limit, page)

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  (page - 1) * limit,
			Sort:  bson.D{{Key: "release_date", Value: -1}, {Key: "_id", Value: -1}},
		}

		filter := bson.M{"published": true}

		albums := []Album{}
		if err := app.DB.FindManyWithOptions(ctx, albumsCollection, filter, opts, &albums); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch recommended albums")
			return
		}

		respondJSON(w, http.StatusOK, albums, "Recommended albums fetched")
	}
}

func GetRecommendations(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		basedOn := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("based_on")))

		filter := bson.M{"published": true}
		sort := bson.D{{Key: "_id", Value: -1}} // default stable sort

		switch basedOn {

		case "recently_played":
			filter["plays"] = bson.M{"$gt": 0}
			sort = bson.D{{Key: "plays", Value: -1}, {Key: "_id", Value: -1}}

		case "language_en":
			filter["language"] = "en"

		case "genre_pop":
			filter["genre"] = "Pop"
		}

		limit, page := getPaginationParams(r)
		limit, page = sanitizePagination(limit, page)

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  (page - 1) * limit,
			Sort:  sort,
		}

		songs := []Song{}
		if err := app.DB.FindManyWithOptions(ctx, songsCollection, filter, opts, &songs); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch recommendations")
			return
		}

		respondJSON(w, http.StatusOK, songs, "Personalized recommendations fetched")
	}
}
