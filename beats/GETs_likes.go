package beats

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"
)

// GetLikers handles GET /likes/:entitytype/users/:entityid
func GetLikers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized: user not found", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")
		if entityType == "" || entityID == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		var likes []models.Like
		err := app.DB.FindMany(
			ctx,
			likesCollection,
			bson.M{
				"entity_type": entityType,
				"entity_id":   entityID,
			},
			&likes,
		)
		if err != nil {
			http.Error(w, "Failed to fetch likers", http.StatusInternalServerError)
			return
		}

		if len(likes) == 0 {
			utils.RespondWithJSON(w, http.StatusOK, map[string]any{
				"likers": []map[string]string{},
			})
			return
		}

		userIDs := make([]string, 0, len(likes))
		for _, like := range likes {
			userIDs = append(userIDs, like.UserID)
		}

		var users []struct {
			UserID   string `bson:"userid"`
			Username string `bson:"username"`
			Avatar   string `bson:"avatar,omitempty"`
		}

		err = app.DB.FindMany(
			ctx,
			usersCollection,
			bson.M{
				"userid": bson.M{"$in": userIDs},
			},
			&users,
		)
		if err != nil {
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}

		userMap := make(map[string]map[string]string, len(users))
		for _, u := range users {
			userMap[u.UserID] = map[string]string{
				"userid":   u.UserID,
				"username": u.Username,
				"avatar":   u.Avatar,
			}
		}

		likers := make([]map[string]string, 0, len(likes))
		for _, like := range likes {
			if meta, ok := userMap[like.UserID]; ok {
				likers = append(likers, meta)
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"likers": likers,
		})
	}
}

// GetLikeCount handles GET /likes/:entitytype/count/:entityid
// GetLikeCount handles GET /likes/:entitytype/count/:entityid
func GetLikeCount(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized: user not found", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")
		if entityType == "" || entityID == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		cacheKey := redisLikeKey(entityType, entityID)

		// 1️⃣ Try cache
		if data, err := app.Cache.Get(ctx, cacheKey); err == nil && len(data) > 0 {
			if count, parseErr := strconv.ParseInt(string(data), 10, 64); parseErr == nil {
				utils.RespondWithJSON(w, http.StatusOK, map[string]int64{
					"count": count,
				})
				return
			}
		}

		// 2️⃣ Fallback to DB
		count, err := app.DB.CountDocuments(
			ctx,
			likesCollection,
			bson.M{
				"entity_type": entityType,
				"entity_id":   entityID,
			},
		)
		if err != nil {
			http.Error(w, "Count failed", http.StatusInternalServerError)
			return
		}

		// 3️⃣ Update cache (best-effort)
		_ = app.Cache.Set(
			ctx,
			cacheKey,
			[]byte(strconv.FormatInt(count, 10)),
			30*time.Second,
		)

		utils.RespondWithJSON(w, http.StatusOK, map[string]int64{
			"count": count,
		})
	}
}
