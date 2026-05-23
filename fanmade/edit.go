package fanmade

import (
	"encoding/json"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"net/http"
	"time"

	"naevis/infra/db"

	"github.com/julienschmidt/httprouter"
)

// EditMedia updates media metadata for a media group
func EditMedia(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")
		mediaID := ps.ByName("id")

		requestingUserID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok || requestingUserID == "" {
			http.Error(w, "Invalid user", http.StatusUnauthorized)
			return
		}

		var payload struct {
			Caption     *string  `json:"caption,omitempty"`
			Description *string  `json:"description,omitempty"`
			Visibility  *string  `json:"visibility,omitempty"`
			Tags        []string `json:"tags,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Fetch the media to get MediaGroupID
		var media models.Media
		if err := app.DB.FindOne(ctx, mediaCollection, map[string]string{
			"entityid":   entityID,
			"entitytype": entityType,
			"mediaid":    mediaID,
		}, &media); err != nil {
			http.Error(w, "Media not found", http.StatusNotFound)
			return
		}

		if media.CreatorID != requestingUserID {
			http.Error(w, "Not authorized to edit this media", http.StatusForbidden)
			return
		}

		// Build update document
		update := map[string]any{"updatedAt": time.Now().UTC()}
		if payload.Caption != nil {
			update["caption"] = *payload.Caption
		}
		if payload.Description != nil {
			update["description"] = *payload.Description
		}
		if payload.Visibility != nil {
			update["visibility"] = *payload.Visibility
		}
		if payload.Tags != nil {
			update["tags"] = payload.Tags
		}

		// Apply update to all media in the same group
		if err := app.DB.UpdateMany(ctx, mediaCollection, map[string]string{"mediaGroupId": media.MediaGroupID}, map[string]any{"$set": update}); err != nil {
			http.Error(w, "Failed to update media group", http.StatusInternalServerError)
			return
		}

		// Fetch updated media group
		var updatedMedias []models.Media
		opts := db.FindManyOptions{} // fetch all
		if err := app.DB.FindManyWithOptions(ctx, mediaCollection, map[string]string{"mediaGroupId": media.MediaGroupID}, opts, &updatedMedias); err != nil {
			http.Error(w, "Failed to fetch updated media", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedMedias)
	}
}
