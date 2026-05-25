package media

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

type FilePayload struct {
	Filename string `json:"filename"`
	Extn     string `json:"extn"`
}

func AddMedia(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")

		if entityType == "" || entityID == "" {
			http.Error(w, "Entity type and ID are required", http.StatusBadRequest)
			return
		}

		requestingUserID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok || requestingUserID == "" {
			http.Error(w, "Invalid or missing user ID", http.StatusUnauthorized)
			return
		}

		var payload struct {
			Caption     string        `json:"caption"`
			CaptionLang string        `json:"captionLang"`
			Files       []FilePayload `json:"files"`
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

		now := time.Now()
		mediaGroupID := "g" + utils.GenerateRandomString(16)

		insertedMedia := make([]models.Media, 0, len(payload.Files))

		for _, file := range payload.Files {
			if file.Filename == "" {
				continue
			}

			// Use provided extension or extract from filename
			extn := file.Extn
			if extn == "" {
				// Try to extract from filename
				if lastDot := strings.LastIndex(file.Filename, "."); lastDot != -1 {
					extn = file.Filename[lastDot:]
				}
			}
			extn = strings.ToLower(extn)

			var mediaType, mimeType string
			switch extn {
			case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif":
				mediaType = models.MediaTypeImage
				mimeType = "image/" + strings.TrimPrefix(extn, ".")
			case ".mp4", ".webm", ".ogg", ".mov", ".avi":
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
				CreatedAt:    now,
				UpdatedAt:    now,
				URL:          file.Filename,
				Extn:         extn,
			}

			if err := app.DB.Insert(ctx, mediaCollection, media); err != nil {
				log.Printf("Failed to insert media %s: %v", file.Filename, err)
				continue
			}

			userdata.SetUserData(
				"media",
				media.MediaID,
				requestingUserID,
				entityType,
				entityID,
				app,
			)

			insertedMedia = append(insertedMedia, media)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(insertedMedia)
	}
}

func DetectCaptionLanguage(caption string) string {
	caption = strings.TrimSpace(caption)
	if caption == "" {
		return "unknown"
	}

	for _, r := range caption {
		switch {
		case r >= 0x4E00 && r <= 0x9FFF:
			return "zh"
		case (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF):
			return "ja"
		case r >= 0xAC00 && r <= 0xD7AF:
			return "ko"
		}
	}

	return "en"
}
