package posts

import (
	"fmt"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"

	"naevis/infra/db"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --- Get single post ---
func GetPost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		postID := ps.ByName("id")

		var post models.BlogPost
		if err := app.DB.FindOne(ctx, blogPostsCollection, map[string]any{
			"postid": postID,
		}, &post); err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Post not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"post": post,
		})
	}
}

func GetAllPosts(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		query := r.URL.Query()

		limit := 20
		page := 1

		if l := query.Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
			if limit > 100 {
				limit = 100
			}
		}
		if p := query.Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
			if page < 1 {
				page = 1
			}
		}

		skip := (page - 1) * limit

		// --- Fetch posts ---
		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var posts []models.BlogPost
		if err := app.DB.FindManyWithOptions(ctx, blogPostsCollection, map[string]any{}, opts, &posts); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch posts")
			return
		}

		if posts == nil {
			posts = []models.BlogPost{}
		}

		// --- Collect unique user IDs ---
		userIDs := make(map[string]struct{})
		for _, p := range posts {
			if p.CreatedBy != "" {
				userIDs[p.CreatedBy] = struct{}{}
			}
		}

		ids := make([]string, 0, len(userIDs))
		for id := range userIDs {
			ids = append(ids, id)
		}

		// --- Fetch usernames ---
		usernames := make(map[string]string)
		if len(ids) > 0 {
			var users []struct {
				UserID   string `bson:"userid"`
				Username string `bson:"username"`
			}

			if err := app.DB.FindMany(ctx, usersCollection, map[string]any{
				"userid_in": ids,
			}, &users); err == nil {
				for _, u := range users {
					usernames[u.UserID] = u.Username
				}
			}
		}

		// --- Build response ---
		resp := make([]models.BlogPostResponse, 0, len(posts))
		for _, p := range posts {
			resp = append(resp, models.BlogPostResponse{
				PostID:      p.PostID,
				Title:       p.Title,
				Category:    p.Category,
				Subcategory: p.Subcategory,
				ReferenceID: p.ReferenceID,
				Thumb:       pickThumb(p.Blocks),
				CreatedBy:   p.CreatedBy,
				Username:    usernames[p.CreatedBy],
				CreatedAt:   p.CreatedAt,
				UpdatedAt:   p.UpdatedAt,
			})
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"posts": resp,
		})
	}
}
