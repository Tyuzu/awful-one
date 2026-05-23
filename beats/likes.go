package beats

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"
)

const likesCollection = "likes"

// redisLikeKey builds the Redis key for a given entityType/entityID.
func redisLikeKey(entityType, entityID string) string {
	return "like:count:" + entityType + ":" + entityID
}

func HowManyLikes(entityType string, entityID string) string {
	return redisLikeKey(entityType, entityID)
}

// ToggleLike handles POST /likes/:entitytype/like/:entityid
func ToggleLike(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized: user not found", http.StatusUnauthorized)
			return
		}

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")

		filter := bson.M{
			"user_id":     userID,
			"entity_type": entityType,
			"entity_id":   entityID,
		}

		redisKey := redisLikeKey(entityType, entityID)

		var existing models.Like
		err := app.DB.FindOne(ctx, likesCollection, filter, &existing)

		// Already liked → unlike
		if err == nil {
			if _, err := app.DB.DeleteOne(ctx, likesCollection, filter); err != nil {
				http.Error(w, "Failed to unlike", http.StatusInternalServerError)
				return
			}

			count := decrementRedisOrMongo(ctx, redisKey, entityType, entityID, app)
			utils.RespondWithJSON(w, http.StatusOK, map[string]any{
				"liked": false,
				"count": count,
			})
			return
		}

		// Not liked → like
		like := models.Like{
			UserID:     userID,
			EntityType: entityType,
			EntityID:   entityID,
			CreatedAt:  time.Now(),
		}

		if err := app.DB.Insert(ctx, likesCollection, like); err != nil {
			http.Error(w, "Failed to like", http.StatusInternalServerError)
			return
		}

		count := incrementRedisOrMongo(ctx, redisKey, entityType, entityID, app)
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"liked": true,
			"count": count,
		})
	}
}

// Cache helpers with DB fallback

func decrementRedisOrMongo(
	ctx context.Context,
	cacheKey,
	entityType,
	entityID string,
	app *infra.Deps,
) int64 {
	// Try cache decrement
	val, err := app.Cache.Incr(ctx, cacheKey)
	if err == nil {
		// Incr only increments, so we simulate decrement by correcting
		val = val - 2 // net effect: -1 from previous value

		if val < 0 {
			_ = app.Cache.Set(
				ctx,
				cacheKey,
				[]byte("0"),
				30*time.Second,
			)
			return 0
		}

		_ = app.Cache.Set(
			ctx,
			cacheKey,
			[]byte(strconv.FormatInt(val, 10)),
			30*time.Second,
		)
		return val
	}

	// Fallback to DB
	count, _ := app.DB.CountDocuments(
		ctx,
		likesCollection,
		bson.M{
			"entity_type": entityType,
			"entity_id":   entityID,
		},
	)
	return count
}

func incrementRedisOrMongo(
	ctx context.Context,
	cacheKey,
	entityType,
	entityID string,
	app *infra.Deps,
) int64 {
	val, err := app.Cache.Incr(ctx, cacheKey)
	if err == nil {
		_ = app.Cache.Set(
			ctx,
			cacheKey,
			[]byte(strconv.FormatInt(val, 10)),
			30*time.Second,
		)
		return val
	}

	// Fallback to DB
	count, _ := app.DB.CountDocuments(
		ctx,
		likesCollection,
		bson.M{
			"entity_type": entityType,
			"entity_id":   entityID,
		},
	)
	return count
}

// BatchUserLikes handles POST /likes/:entitytype/batch/users
func BatchUserLikes(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized: user not found", http.StatusUnauthorized)
			return
		}

		var req struct {
			EntityIDs []string `json:"entity_ids"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if len(req.EntityIDs) == 0 {
			utils.RespondWithJSON(w, http.StatusOK, map[string]any{
				"data": map[string]bool{},
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var likes []models.Like
		err := app.DB.FindMany(
			ctx,
			likesCollection,
			bson.M{
				"user_id":   userID,
				"entity_id": bson.M{"$in": req.EntityIDs},
			},
			&likes,
		)
		if err != nil {
			http.Error(w, "Failed to query likes", http.StatusInternalServerError)
			return
		}

		likedSet := make(map[string]struct{}, len(likes))
		for _, like := range likes {
			likedSet[like.EntityID] = struct{}{}
		}

		result := make(map[string]bool, len(req.EntityIDs))
		for _, eid := range req.EntityIDs {
			_, liked := likedSet[eid]
			result[eid] = liked
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"data": result,
		})
	}
}
