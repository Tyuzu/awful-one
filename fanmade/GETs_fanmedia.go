package fanmade

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// GetMedia returns a single media item by ID
func GetMedia(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")
		mediaID := ps.ByName("id")

		var media models.Media
		err := app.DB.FindOne(r.Context(), mediaCollection, map[string]string{
			"entityid":   entityID,
			"entitytype": entityType,
			"mediaid":    mediaID,
		}, &media)
		if err != nil {
			http.Error(w, "Media not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(media)
	}
}

// GetMedias returns all media for an entity
func GetMedias(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := map[string]string{
			"entityid":   ps.ByName("entityid"),
			"entitytype": ps.ByName("entitytype"),
		}

		var medias []models.Media
		opts := db.FindManyOptions{} // no pagination
		if err := app.DB.FindManyWithOptions(ctx, mediaCollection, filter, opts, &medias); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve media")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, medias)
	}
}

// GetMediaGroups returns media grouped by MediaGroupID
func GetMediaGroups(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := map[string]string{
			"entityid":   ps.ByName("entityid"),
			"entitytype": ps.ByName("entitytype"),
		}

		var medias []models.Media
		opts := db.FindManyOptions{} // retrieve all
		if err := app.DB.FindManyWithOptions(ctx, mediaCollection, filter, opts, &medias); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve media")
			return
		}

		mediaMap := make(map[string][]models.Media)
		for _, m := range medias {
			mediaMap[m.MediaGroupID] = append(mediaMap[m.MediaGroupID], m)
		}

		// convert map to slice of groups
		groups := make([]map[string]any, 0, len(mediaMap))
		for groupID, medias := range mediaMap {
			groups = append(groups, map[string]any{
				"groupId": groupID,
				"files":   medias,
			})
		}

		utils.RespondWithJSON(w, http.StatusOK, groups)
	}
}
