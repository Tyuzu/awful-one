package feed

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type BulkMetadataRequest struct {
	IDs []string `json:"ids"`
}

type PostMetadata struct {
	PostID      string `json:"postId"`
	Likes       int64  `json:"likes"`
	Comments    int64  `json:"comments"`
	LikedByUser bool   `json:"likedByUser"`
}

func GetPostsMetadata(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		var req BulkMetadataRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		userID, _ := r.Context().Value("userId").(string)
		postIDs := req.IDs

		// --- Aggregate likes ---
		likePipeline := []any{
			map[string]any{"$match": map[string]any{"postid": map[string]any{"$in": postIDs}}},
			map[string]any{"$group": map[string]any{"_id": "$postid", "count": map[string]any{"$sum": 1}}},
		}

		var likeResults []struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := app.DB.Aggregate(ctx, likesCollection, likePipeline, &likeResults); err != nil {
			http.Error(w, "Failed to aggregate likes", http.StatusInternalServerError)
			return
		}

		likeCounts := make(map[string]int64, len(likeResults))
		for _, r := range likeResults {
			likeCounts[r.ID] = r.Count
		}

		// --- Aggregate comments ---
		commentPipeline := []any{
			map[string]any{"$match": map[string]any{"postid": map[string]any{"$in": postIDs}}},
			map[string]any{"$group": map[string]any{"_id": "$postid", "count": map[string]any{"$sum": 1}}},
		}

		var commentResults []struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := app.DB.Aggregate(ctx, commentsCollection, commentPipeline, &commentResults); err != nil {
			http.Error(w, "Failed to aggregate comments", http.StatusInternalServerError)
			return
		}

		commentCounts := make(map[string]int64, len(commentResults))
		for _, r := range commentResults {
			commentCounts[r.ID] = r.Count
		}

		// --- Find which posts are liked by the user ---
		likedByUser := make(map[string]bool)
		if userID != "" {
			filter := map[string]any{
				"postid": map[string]any{"$in": postIDs},
				"userid": userID,
			}
			var userLikes []struct {
				PostID string `bson:"postid"`
			}
			if err := app.DB.FindMany(ctx, likesCollection, filter, &userLikes); err == nil {
				for _, l := range userLikes {
					likedByUser[l.PostID] = true
				}
			}
		}

		// --- Assemble final response ---
		result := make([]PostMetadata, 0, len(postIDs))
		for _, pid := range postIDs {
			result = append(result, PostMetadata{
				PostID:      pid,
				Likes:       likeCounts[pid],
				Comments:    commentCounts[pid],
				LikedByUser: likedByUser[pid],
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
