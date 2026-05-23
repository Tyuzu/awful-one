package media

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetMedia(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")
		mediaID := ps.ByName("id")

		var media models.Media
		err := app.DB.FindOne(ctx, mediaCollection, bson.M{
			"entityid":   entityID,
			"entitytype": entityType,
			"mediaid":    mediaID,
		}, &media)
		if err != nil {
			http.Error(w, "Media not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(media)
	}
}

// ---------------------- Get Medias ----------------------
func GetMedias(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"entityid":   ps.ByName("entityid"),
			"entitytype": ps.ByName("entitytype"),
		}

		var medias []models.Media
		err := app.DB.FindMany(ctx, mediaCollection, filter, &medias)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve media")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, medias)
	}
}

// ---------------------- Get Media Groups ----------------------
func GetMediaGroups(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"entityid":   ps.ByName("entityid"),
			"entitytype": ps.ByName("entitytype"),
		}

		var medias []models.Media
		err := app.DB.FindMany(ctx, mediaCollection, filter, &medias)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve media")
			return
		}

		mediaMap := make(map[string][]models.Media)
		for _, m := range medias {
			mediaMap[m.MediaGroupID] = append(mediaMap[m.MediaGroupID], m)
		}

		groups := make([]map[string]any, 0, len(mediaMap))
		for groupID, files := range mediaMap {
			groups = append(groups, map[string]any{
				"groupId": groupID,
				"files":   files,
			})
		}

		utils.RespondWithJSON(w, http.StatusOK, groups)
	}
}
