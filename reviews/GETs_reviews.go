package reviews

import (
	"context"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* -------------------------
   Get Reviews (list)
------------------------- */

func GetReviews(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		entityType := ps.ByName("entityType")
		entityId := ps.ByName("entityId")

		skip, limit := utils.ParsePagination(r, 10, 100)

		filter := bson.M{
			"entityType": entityType,
			"entityId":   entityId,
		}

		var reviews []models.Review
		if err := app.DB.FindMany(ctx, reviewsCollection, filter, &reviews); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch reviews"})
			return
		}

		utils.SortAndSlice(
			&reviews,
			bson.D{{Key: "createdAt", Value: -1}},
			int64(skip),
			int64(limit),
		)

		if reviews == nil {
			reviews = []models.Review{}
		}

		respond(w, http.StatusOK, reviews)
	}
}

/* -------------------------
   Get Review (single)
------------------------- */

func GetReview(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		reviewId := ps.ByName("reviewId")

		var review models.Review
		if err := app.DB.FindOne(
			r.Context(),
			reviewsCollection,
			bson.M{"reviewid": reviewId},
			&review,
		); err != nil {
			respond(w, http.StatusNotFound, map[string]string{"error": "Review not found"})
			return
		}

		respond(w, http.StatusOK, review)
	}
}
