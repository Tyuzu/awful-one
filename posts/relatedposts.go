package posts

import (
	"context"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"time"

	"naevis/infra/db"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

type Post struct {
	PostID      string   `bson:"postid" json:"postid"`
	Title       string   `bson:"title" json:"title"`
	Category    string   `bson:"category" json:"category"`
	Subcategory string   `bson:"subcategory" json:"subcategory"`
	Tags        []string `bson:"tags" json:"tags"`
	CreatedAt   int64    `bson:"createdAt" json:"createdAt"`
	CreatedBy   string   `bson:"createdBy" json:"createdBy"`
}

func GetRelatedPosts(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		postID := r.URL.Query().Get("postid")
		category := r.URL.Query().Get("category")
		subcategory := r.URL.Query().Get("subcategory")
		tags := r.URL.Query()["tags"]

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Base filter
		filter := map[string]any{
			"postid_ne": postID, // uses your translateFilter(_ne)
			"$or": []any{
				map[string]any{"category": category},
				map[string]any{"subcategory": subcategory},
			},
		}

		// Optional tags match
		if len(tags) > 0 {
			filter["$or"] = append(
				filter["$or"].([]any),
				map[string]any{"tags": map[string]any{"$in": tags}},
			)
		}

		opts := db.FindManyOptions{
			Limit: 10,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var related []Post
		if err := app.DB.FindManyWithOptions(ctx, blogPostsCollection, filter, opts, &related); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if related == nil {
			related = []Post{}
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"related": related,
		})
	}
}
