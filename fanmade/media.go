package fanmade

import (
	"encoding/json"
	"log"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

// AddMedia handles uploading media for an entity
func AddMedia(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")

		if entityID == "" {
			http.Error(w, "Entity ID is required", http.StatusBadRequest)
			return
		}

		requestingUserID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok || requestingUserID == "" {
			http.Error(w, "Invalid or missing user ID", http.StatusUnauthorized)
			return
		}

		var payload struct {
			Caption     string                   `json:"caption"`
			CaptionLang string                   `json:"captionLang"`
			Files       []map[string]interface{} `json:"files"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(payload.Files) == 0 {
			http.Error(w, "No files provided", http.StatusBadRequest)
			return
		}

		lang := payload.CaptionLang
		if lang == "" || lang == "unknown" {
			lang = DetectCaptionLanguage(payload.Caption)
		}

		mediaGroupID := "g" + utils.GenerateRandomString(16)
		var insertedMedia []models.Media

		for _, fileData := range payload.Files {
			filename, _ := fileData["filename"].(string)
			if filename == "" {
				continue
			}

			extn, _ := fileData["extn"].(string)
			if extn == "" {
				// Try to extract from filename
				if lastDot := strings.LastIndex(filename, "."); lastDot != -1 {
					extn = filename[lastDot:]
				}
			}
			extn = strings.ToLower(extn)

			var mediaType, mimeType string
			switch extn {
			case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif":
				mediaType = models.MediaTypeImage
				mimeType = "image/" + strings.TrimPrefix(extn, ".")
			case ".mp4", ".webm", ".ogg", ".mov", ".avi":
				mediaType = models.MediaTypeVideo
				mimeType = "video/" + strings.TrimPrefix(extn, ".")
			default:
				mediaType = "unknown"
				mimeType = "application/octet-stream"
			}

			media := models.Media{
				MediaID:      "m" + utils.GenerateRandomString(16),
				MediaGroupID: mediaGroupID,
				EntityID:     entityID,
				EntityType:   entityType,
				Type:         mediaType,
				MimeType:     mimeType,
				Caption:      payload.Caption,
				CaptionLang:  lang,
				CreatorID:    requestingUserID,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
				URL:          filename,
				Extn:         extn,
			}

			if err := app.DB.Insert(ctx, mediaCollection, media); err != nil {
				log.Printf("Failed to insert media %s: %v", filename, err)
				continue
			}

			userdata.SetUserData("media", media.MediaID, requestingUserID, entityType, entityID, app)
			insertedMedia = append(insertedMedia, media)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(insertedMedia)
	}
}

// DetectCaptionLanguage detects the language of a caption
func DetectCaptionLanguage(caption string) string {
	caption = strings.TrimSpace(caption)
	if caption == "" {
		return "unknown"
	}

	for _, r := range caption {
		switch {
		case r >= 0x4E00 && r <= 0x9FFF:
			return "zh" // Chinese
		case (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF):
			return "ja" // Japanese
		case r >= 0xAC00 && r <= 0xD7AF:
			return "ko" // Korean
		}
	}

	return "en"
}
