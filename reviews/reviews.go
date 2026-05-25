package reviews

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* -------------------------
   Helpers
------------------------- */

func respond(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func getUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(globals.UserIDKey).(string)
	return id, ok && id != ""
}

func isAdmin(ctx context.Context) bool {
	role, ok := ctx.Value("role").(string)
	return ok && role == "admin"
}

/* -------------------------
   Payloads
------------------------- */

type CreateReviewPayload struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type UpdateReviewPayload struct {
	Rating  *int   `json:"rating,omitempty"`
	Comment string `json:"comment,omitempty"`
}

/* -------------------------
   Add Review
------------------------- */

func AddReview(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userId, ok := getUserID(r.Context())
		if !ok {
			respond(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}

		entityType := ps.ByName("entityType")
		entityId := ps.ByName("entityId")

		dupFilter := bson.M{
			"userid":     userId,
			"entityType": entityType,
			"entityId":   entityId,
		}

		var existing models.Review
		if err := app.DB.FindOne(r.Context(), reviewsCollection, dupFilter, &existing); err == nil {
			respond(w, http.StatusConflict, map[string]string{"error": "Already reviewed"})
			return
		}

		var payload CreateReviewPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respond(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
			return
		}

		if payload.Rating < 1 || payload.Rating > 5 || payload.Comment == "" {
			respond(w, http.StatusBadRequest, map[string]string{"error": "Invalid review data"})
			return
		}

		now := time.Now().UTC()
		review := models.Review{
			ReviewID:   utils.GenerateRandomString(16),
			UserID:     userId,
			EntityType: entityType,
			EntityID:   entityId,
			Rating:     payload.Rating,
			Comment:    payload.Comment,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := app.DB.Insert(r.Context(), reviewsCollection, review); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create review"})
			return
		}

		respond(w, http.StatusCreated, review)
	}
}

/* -------------------------
   Edit Review
------------------------- */

func EditReview(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userId, ok := getUserID(r.Context())
		if !ok {
			respond(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}

		reviewId := ps.ByName("reviewId")

		var existing models.Review
		if err := app.DB.FindOne(
			r.Context(),
			reviewsCollection,
			bson.M{"reviewid": reviewId},
			&existing,
		); err != nil {
			respond(w, http.StatusNotFound, map[string]string{"error": "Review not found"})
			return
		}

		if existing.UserID != userId && !isAdmin(r.Context()) {
			respond(w, http.StatusForbidden, map[string]string{"error": "Forbidden"})
			return
		}

		var payload UpdateReviewPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respond(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
			return
		}

		update := bson.M{
			"updatedAt": time.Now().UTC(),
		}

		if payload.Rating != nil {
			if *payload.Rating < 1 || *payload.Rating > 5 {
				respond(w, http.StatusBadRequest, map[string]string{"error": "Invalid rating"})
				return
			}
			update["rating"] = *payload.Rating
		}

		if payload.Comment != "" {
			update["comment"] = payload.Comment
		}

		if len(update) == 1 {
			respond(w, http.StatusBadRequest, map[string]string{"error": "Nothing to update"})
			return
		}

		if err := app.DB.Update(
			r.Context(),
			reviewsCollection,
			bson.M{"reviewid": reviewId},
			update,
		); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update review"})
			return
		}

		respond(w, http.StatusOK, map[string]string{"message": "Review updated"})
	}
}

/* -------------------------
   Delete Review
------------------------- */

func DeleteReview(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userId, ok := getUserID(r.Context())
		if !ok {
			respond(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}

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

		if review.UserID != userId && !isAdmin(r.Context()) {
			respond(w, http.StatusForbidden, map[string]string{"error": "Forbidden"})
			return
		}

		if _, err := app.DB.Delete(
			r.Context(),
			reviewsCollection,
			bson.M{"reviewid": reviewId},
		); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete review"})
			return
		}

		respond(w, http.StatusOK, map[string]string{"message": "Review deleted"})
	}
}
