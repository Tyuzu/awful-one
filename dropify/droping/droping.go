package droping

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"

	"naevis/config/mqevent"
	"naevis/dropify/filemgr"
	"naevis/dropify/services"
	"naevis/infra"
	"naevis/utils"
)

const maxUploadBytes = 200 << 20 // 200 MB

// Attachment represents a file attachment in responses
type Attachment struct {
	Filename    string `json:"filename"`
	Extension   string `json:"extension"`
	Key         string `json:"key"`
	Resolutions []int  `json:"resolutions,omitempty"`
}

// valid entity types
var validEntities = map[string]filemgr.EntityType{
	"artist":       filemgr.EntityArtist,
	"user":         filemgr.EntityUser,
	"baito":        filemgr.EntityBaito,
	"baito_worker": filemgr.EntityWorker,
	"song":         filemgr.EntitySong,
	"post":         filemgr.EntityPost,
	"chat":         filemgr.EntityChat,
	"event":        filemgr.EntityEvent,
	"farm":         filemgr.EntityFarm,
	"crop":         filemgr.EntityCrop,
	"place":        filemgr.EntityPlace,
	"media":        filemgr.EntityMedia,
	"feedpost":     filemgr.EntityFeed,
	"recipe":       filemgr.EntityRecipe,
	"product":      filemgr.EntityProduct,
	"live":         filemgr.EntityLive,
}

// FiledropHandler handles file uploads via multipart/form-data
func FiledropHandler(
	app *infra.Deps,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {

	// -------------------------
	// Validate request
	// -------------------------

	if err := validateUploadRequest(w, r); err != nil {

		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

		return
	}

	// -------------------------
	// Parse multipart form
	// -------------------------

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {

		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			"failed to parse multipart form: "+err.Error(),
		)

		return
	}

	// cleanup temp files
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	// -------------------------
	// Frontend fields
	// -------------------------

	entityType := strings.ToLower(
		strings.TrimSpace(r.FormValue("entityType")),
	)

	entityId := strings.TrimSpace(
		r.FormValue("entityId"),
	)

	remoteURL := strings.TrimSpace(
		r.FormValue("remoteUrl"),
	)

	remoteKey := strings.TrimSpace(
		r.FormValue("remoteKey"),
	)

	// -------------------------
	// Validate entity type
	// -------------------------

	if entityType == "" {

		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			"entityType is required",
		)

		return
	}

	if _, ok := validEntities[entityType]; !ok {

		utils.RespondWithError(
			w,
			http.StatusBadRequest,
			"invalid entityType",
		)

		return
	}

	log.Printf(
		"[Filedrop] entityType=%s entityId=%s",
		entityType,
		entityId,
	)

	// -------------------------
	// Service
	// -------------------------

	fileService := services.NewFileService()

	var (
		attachments []services.Attachment
		err         error
	)

	// -------------------------
	// Remote URL upload
	// -------------------------

	if remoteURL != "" {

		switch remoteKey {

		case "banner", "photo", "avatar", "seating":

		default:

			utils.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid remoteKey",
			)

			return
		}

		attachments, err = fileService.ProcessRemoteFile(
			remoteURL,
			remoteKey,
			entityType,
			entityId,
		)

	} else {

		// -------------------------
		// Multipart upload
		// -------------------------

		if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {

			utils.RespondWithError(
				w,
				http.StatusBadRequest,
				"no files uploaded",
			)

			return
		}

		attachments, err = fileService.ProcessUploadedFiles(
			r,
			entityType,
			entityId,
		)
	}

	// -------------------------
	// Handle processing errors
	// -------------------------

	if err != nil {

		log.Printf(
			"[Filedrop] processing error: %v",
			err,
		)

		utils.RespondWithError(
			w,
			http.StatusInternalServerError,
			"failed to process files: "+err.Error(),
		)

		return
	}

	// -------------------------
	// Publish media events
	// -------------------------

	for _, attachment := range attachments {

		mediaPayload := mqevent.MediaUploadedPayload{
			EntityType: entityType,
			EntityID:   entityId,
			Extension:  attachment.Extension,
			FileName:   attachment.Filename,
			FilePath:   attachment.Key,
			Timestamp:  time.Now().UTC().UnixNano(),
		}

		mediaBytes, err := json.Marshal(mediaPayload)
		if err != nil {

			log.Printf(
				"[Filedrop] failed to marshal media event: %v",
				err,
			)

			continue
		}

		publishCtx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)

		err = app.MQ.Publish(
			publishCtx,
			mqevent.MediaUploaded,
			mediaBytes,
		)

		cancel()

		if err != nil {

			log.Printf(
				"[Filedrop] failed to publish media event: %v",
				err,
			)

			continue
		}

		log.Printf(
			"[Filedrop] published media event entity=%s id=%s file=%s path=%s",
			entityType,
			entityId,
			attachment.Filename,
			attachment.Key,
		)
	}

	// -------------------------
	// Response
	// -------------------------

	response := convertToAttachments(attachments)

	utils.RespondWithJSON(
		w,
		http.StatusOK,
		response,
	)
}

// validateUploadRequest validates upload request basics
func validateUploadRequest(
	w http.ResponseWriter,
	r *http.Request,
) error {

	// limit body size
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxUploadBytes,
	)

	// only POST allowed
	if r.Method != http.MethodPost {
		return fmt.Errorf("method must be POST")
	}

	contentType := r.Header.Get("Content-Type")

	// remote uploads may not use multipart
	remoteURL := strings.TrimSpace(
		r.FormValue("remoteUrl"),
	)

	if remoteURL == "" &&
		!strings.HasPrefix(contentType, "multipart/") {

		return fmt.Errorf(
			"content-type must be multipart/form-data",
		)
	}

	return nil
}

// convertToAttachments converts service attachments to response format
func convertToAttachments(
	serviceAttachments []services.Attachment,
) []Attachment {

	attachments := make(
		[]Attachment,
		len(serviceAttachments),
	)

	for i, sa := range serviceAttachments {

		attachments[i] = Attachment{
			Filename:    sa.Filename,
			Extension:   sa.Extension,
			Key:         sa.Key,
			Resolutions: sa.Resolutions,
		}
	}

	return attachments
}

// OptionsHandler handles CORS preflight requests
func OptionsHandler(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {

	w.Header().Set(
		"Access-Control-Allow-Origin",
		"*",
	)

	w.Header().Set(
		"Access-Control-Allow-Methods",
		"GET, POST, PUT, DELETE, OPTIONS",
	)

	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Content-Type, Authorization",
	)

	w.WriteHeader(http.StatusNoContent)
}
