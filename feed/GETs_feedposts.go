package feed

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/models"
	"net/http"
	"strconv"
	"time"

	"naevis/infra/db"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// GetPost returns a single post enriched with like count
func GetPost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("postid")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Step 1: Fetch post from DB
		var post models.FeedPost
		if err := app.DB.FindOne(ctx, feedpostsCollection, map[string]any{"postid": id}, &post); err != nil {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}

		// Step 2: Fetch like count from Cache
		redisKey := "like:count:post:" + id
		var likeCount int64
		if data, err := app.Cache.Get(ctx, redisKey); err == nil && data != nil {
			likeCount, _ = strconv.ParseInt(string(data), 10, 64)
		} else {
			// fallback to DB count
			likeCount, _ = app.DB.CountDocuments(ctx, likesCollection, map[string]any{
				"entity_type": "post",
				"entity_id":   id,
			})
			// cache the result
			_ = app.Cache.Set(ctx, redisKey, []byte(strconv.FormatInt(likeCount, 10)), 10*time.Minute)
		}

		post.Likes = likeCount

		// Step 3: Update DB with latest like count
		_ = app.DB.UpdateOne(ctx, feedpostsCollection, map[string]any{"postid": id}, map[string]any{"likes": likeCount})

		// Step 4: Return enriched post
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(post); err != nil {
			http.Error(w, "Failed to encode post data", http.StatusInternalServerError)
		}
	}
}

// GetPosts returns a list of posts with usernames populated from Cache
func GetPosts(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		opts := db.FindManyOptions{
			Limit: 100,
			Sort:  bson.D{{Key: "timestamp", Value: -1}},
			Skip:  0,
		}
		var posts []models.FeedPost
		if err := app.DB.FindManyWithOptions(ctx, feedpostsCollection, map[string]any{}, opts, &posts); err != nil {
			http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
			return
		}

		if len(posts) == 0 {
			posts = []models.FeedPost{}
		}

		// collect unique user IDs
		userIDs := map[string]struct{}{}
		for _, p := range posts {
			userIDs[p.UserID] = struct{}{}
		}

		// fetch usernames from Cache
		usernameMap := map[string]string{}
		for id := range userIDs {
			if data, err := app.Cache.HGet(ctx, "users", id); err == nil && data != nil {
				usernameMap[id] = string(data)
			} else {
				usernameMap[id] = "unknown"
			}
		}

		// populate posts
		for i := range posts {
			if uname, ok := usernameMap[posts[i].UserID]; ok && uname != "" {
				posts[i].Username = uname
			} else if posts[i].Username == "" {
				posts[i].Username = "unknown"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":   true,
			"data": posts,
		})
	}
}
