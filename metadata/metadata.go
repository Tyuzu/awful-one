package metadata

import (
	"context"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// GetUsersMeta returns minimal metadata for a set of users
func GetUsersMeta(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		idsParam := r.URL.Query().Get("ids")
		if idsParam == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing ids param")
			return
		}

		ids := strings.Split(idsParam, ",")
		filter := bson.M{
			"userid": bson.M{"$in": ids},
		}

		// Fetch full user documents (projection not supported by interface)
		var users []struct {
			UserID   string `bson:"userid"`
			Username string `bson:"username"`
			Name     string `bson:"name"`
			Avatar   string `bson:"avatar"`
		}

		err := app.DB.FindMany(ctx, usersCollection, filter, &users)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "DB query failed")
			return
		}

		result := make(map[string]map[string]string, len(users))
		for _, user := range users {
			result[user.UserID] = map[string]string{
				"username": user.Username,
				"name":     user.Name,
				"avatar":   user.Avatar,
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, result)
	}
}
