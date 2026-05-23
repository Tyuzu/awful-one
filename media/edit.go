package media

import (
	"encoding/json"
	"net/http"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

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

		// Fetch media to verify existence and ownership
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

		// Authorization check
		if media.CreatorID != requestingUserID {
			http.Error(w, "Not authorized to edit this media", http.StatusForbidden)
			return
		}

		// Build update document
		updateFields := bson.M{
			"updatedAt": time.Now(),
		}

		if payload.Caption != nil {
			updateFields["caption"] = *payload.Caption
		}
		if payload.Description != nil {
			updateFields["description"] = *payload.Description
		}
		if payload.Visibility != nil {
			updateFields["visibility"] = *payload.Visibility
		}
		if payload.Tags != nil {
			updateFields["tags"] = payload.Tags
		}

		// Update all media in the same group
		err = app.DB.UpdateMany(ctx, mediaCollection,
			bson.M{"mediaGroupId": media.MediaGroupID},
			bson.M{"$set": updateFields},
		)
		if err != nil {
			http.Error(w, "Failed to update media group", http.StatusInternalServerError)
			return
		}

		// Fetch updated media group
		var updatedMedias []models.Media
		err = app.DB.FindMany(ctx, mediaCollection,
			bson.M{"mediaGroupId": media.MediaGroupID},
			&updatedMedias,
		)
		if err != nil {
			http.Error(w, "Failed to load updated media", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updatedMedias)
	}
}
