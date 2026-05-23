package comments

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"
)

/* =========================
   GET SINGLE COMMENT
========================= */

func GetComment(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		commentID := ps.ByName("commentid")
		if commentID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid comment ID")
			return
		}

		var comment models.Comment
		err := app.DB.FindOne(
			ctx,
			commentsCollection,
			bson.M{"commentid": commentID},
			&comment,
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Comment not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, comment)
	}
}

/*
	=========================
	  GET COMMENTS (PAGINATED)

=========================
*/
func GetComments(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")

		if entityID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Entity ID is required")
			return
		}

		if !isValidEntityType(entityType) {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid entity type")
			return
		}

		/* ---------- Pagination ---------- */
		page := 1
		limit := 10

		if v := r.URL.Query().Get("page"); v != "" {
			if p, err := strconv.Atoi(v); err == nil && p > 0 {
				page = p
			}
		}

		if v := r.URL.Query().Get("limit"); v != "" {
			if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 50 {
				limit = l
			}
		}

		skip := (page - 1) * limit
		sortBy := r.URL.Query().Get("sort") // new | old | likes

		filter := bson.M{
			"entity_type": entityType,
			"entity_id":   entityID,
		}

		/* ---------- Sorting (ORDERED) ---------- */
		sort := bson.D{
			{Key: "created_at", Value: -1},
			{Key: "commentid", Value: -1},
		}

		switch sortBy {
		case "old":
			sort = bson.D{
				{Key: "created_at", Value: 1},
				{Key: "commentid", Value: 1},
			}
		case "likes":
			sort = bson.D{
				{Key: "likes", Value: -1},
				{Key: "created_at", Value: -1},
			}
		}

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort:  sort,
		}

		var comments []models.Comment
		if err := app.DB.FindManyWithOptions(ctx, commentsCollection, filter, opts, &comments); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch comments")
			return
		}

		// Always return array (never null)
		if comments == nil {
			comments = []models.Comment{}
		}

		utils.RespondWithJSON(w, http.StatusOK, comments)
	}
}
