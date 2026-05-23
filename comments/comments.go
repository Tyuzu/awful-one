package comments

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"
)

/* =========================
   HELPERS
========================= */

func isValidEntityType(t string) bool {
	switch t {
	case "post", "article", "profile", "recipe": // Added "recipe"
		return true
	default:
		return false
	}
}

/* =========================
   CREATE
========================= */

func CreateComment(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")

		if !isValidEntityType(entityType) {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid entity type")
			return
		}

		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		content := strings.TrimSpace(body.Content)
		if content == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Comment cannot be empty")
			return
		}

		comment := models.Comment{
			CommentID:  utils.GenerateRandomString(18),
			EntityType: entityType,
			EntityID:   entityID,
			Content:    content,
			CreatedBy:  utils.GetUserIDFromRequest(r),
			Likes:      0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := app.DB.Insert(ctx, commentsCollection, comment); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "DB insert failed")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, comment)
	}
}

/* =========================
   UPDATE
========================= */

func UpdateComment(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		commentID := ps.ByName("commentid")
		if commentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID")
			return
		}

		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		content := strings.TrimSpace(body.Content)
		if content == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Comment cannot be empty")
			return
		}

		/* Fetch + ownership check */
		var existing models.Comment
		err := app.DB.FindOne(
			ctx,
			commentsCollection,
			bson.M{"commentid": commentID},
			&existing,
		)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				utils.RespondWithError(w, http.StatusNotFound, "Comment not found")
				return
			}
			utils.RespondWithError(w, http.StatusInternalServerError, "DB error")
			return
		}

		if existing.CreatedBy != utils.GetUserIDFromRequest(r) {
			utils.RespondWithError(w, http.StatusForbidden, "Forbidden")
			return
		}

		update := bson.M{
			"content":    content,
			"updated_at": time.Now(),
		}

		if err := app.DB.UpdateOne(
			ctx,
			commentsCollection,
			bson.M{"commentid": commentID},
			update,
		); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "DB update failed")
			return
		}

		/* Return updated document */
		err = app.DB.FindOne(
			ctx,
			commentsCollection,
			bson.M{"commentid": commentID},
			&existing,
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Fetch failed")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, existing)
	}
}

/* =========================
   DELETE
========================= */

func DeleteComment(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		commentID := ps.ByName("commentid")
		if commentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID")
			return
		}

		filter := bson.M{
			"commentid": commentID,
			"createdby": utils.GetUserIDFromRequest(r),
		}

		_, err := app.DB.Delete(ctx, commentsCollection, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				utils.RespondWithError(w, http.StatusForbidden, "Comment not found or forbidden")
				return
			}
			utils.RespondWithError(w, http.StatusInternalServerError, "Delete failed")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
