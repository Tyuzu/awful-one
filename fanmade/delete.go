package fanmade

import (
	"encoding/json"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// DeleteMedia deletes a single media item if the requesting user is the creator
func DeleteMedia(app *infra.Deps) httprouter.Handle {
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

		// Fetch the media using Database interface
		var media models.Media
		err := app.DB.FindOne(ctx, mediaCollection, map[string]string{
			"entityid":   entityID,
			"entitytype": entityType,
			"mediaid":    mediaID,
		}, &media)
		if err != nil {
			http.Error(w, "Media not found", http.StatusNotFound)
			return
		}

		if media.CreatorID != requestingUserID {
			http.Error(w, "Not authorized to delete this media", http.StatusForbidden)
			return
		}

		// Delete media using Database interface
		if _, err := app.DB.DeleteOne(ctx, mediaCollection, map[string]string{"mediaid": mediaID}); err != nil {
			http.Error(w, "Failed to delete media", http.StatusInternalServerError)
			return
		}

		userdata.DelUserData("media", mediaID, requestingUserID, app)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Media deleted successfully",
		})
	}
}
