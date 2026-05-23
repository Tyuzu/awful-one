package vlive

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ----------------------- COMMON HELPERS -----------------------

// fetchStream fetches a stream by hex ID, optionally enforcing ownership
func fetchStream(ctx context.Context, app *infra.Deps, idHex, userID string, requireOwner bool) (models.LiveStream, error) {
	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return models.LiveStream{}, errors.New("invalid liveid")
	}

	stream, err := getStreamByID(ctx, app, id)
	if err != nil {
		return models.LiveStream{}, errors.New("stream not found")
	}

	if requireOwner && !isOwner(userID, stream) {
		return models.LiveStream{}, errors.New("forbidden")
	}

	return stream, nil
}

// validate metadata for updates
func validateMetadata(payload map[string]interface{}) (bson.M, error) {
	update := bson.M{}

	if title, ok := payload["title"].(string); ok {
		title = strings.TrimSpace(title)
		if title == "" {
			return nil, errors.New("title cannot be empty")
		}
		update["title"] = title
	}

	if desc, ok := payload["description"].(string); ok {
		update["description"] = desc
	}

	if thumb, ok := payload["thumbnail"].(string); ok {
		update["thumbnail"] = thumb
	}

	if tags, ok := payload["tags"].([]interface{}); ok {
		clean := []string{}
		for _, t := range tags {
			if s, ok := t.(string); ok {
				clean = append(clean, s)
			}
		}
		update["tags"] = clean
	}

	if slow, ok := payload["slowModeSeconds"].(float64); ok {
		if slow < 0 || slow > 3600 {
			return nil, errors.New("slowModeSeconds must be between 0 and 3600")
		}
		update["slow_mode_seconds"] = int(slow)
	}

	return update, nil
}

// ----------------------- STATE MACHINE -----------------------

var validTransitions = map[string][]string{
	models.LiveCreated:   {models.LiveReady, models.LiveCancelled},
	models.LiveReady:     {models.LiveLive, models.LiveCancelled},
	models.LiveLive:      {models.LiveEnded, models.LiveError},
	models.LiveError:     {models.LiveEnded, models.LiveCancelled},
	models.LiveCancelled: {}, // terminal
	models.LiveEnded:     {}, // terminal
}

func CanTransition(from, to string) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, v := range allowed {
		if v == to {
			return true
		}
	}
	return false
}

func EnforceTransition(from, to string) error {
	if !CanTransition(from, to) {
		return errors.New("invalid livestream state transition")
	}
	return nil
}

// ----------------------- STREAM HANDLERS -----------------------

func CreateStream(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		var payload struct {
			EntityType      string   `json:"entityType"`
			EntityID        string   `json:"entityId"`
			Title           string   `json:"title"`
			Description     *string  `json:"description"`
			IsPublic        *bool    `json:"isPublic"`
			Thumbnail       *string  `json:"thumbnail"`
			Tags            []string `json:"tags"`
			SlowModeSeconds *int     `json:"slowModeSeconds"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, "invalid body", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(payload.Title) == "" {
			writeError(w, "title is required", http.StatusBadRequest)
			return
		}
		if !validateEntityType(payload.EntityType) {
			writeError(w, "invalid entityType", http.StatusBadRequest)
			return
		}
		if payload.EntityID == "" {
			writeError(w, "entityId required", http.StatusBadRequest)
			return
		}

		// 🔒 entity access check (creator must own or be member)
		if !CheckEntityAccess(
			r.Context(),
			app,
			userID,
			payload.EntityType,
			payload.EntityID,
		) {
			writeError(w, "forbidden", http.StatusForbidden)
			return
		}

		activeFilter := bson.M{
			"creator_id": userID,
			"state": bson.M{
				"$in": []string{
					models.LiveCreated,
					models.LiveReady,
					models.LiveLive,
				},
			},
		}

		count, err := app.DB.CountDocuments(r.Context(), vlivesCollection, activeFilter)
		if err != nil {
			log.Printf("CreateStream: count error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			writeError(w, "creator already has an active livestream", http.StatusConflict)
			return
		}

		isPublic := false
		if payload.IsPublic != nil {
			isPublic = *payload.IsPublic
		}

		stream := models.LiveStream{
			LiveID:          utils.GenerateRandomDigitString(14),
			EntityType:      payload.EntityType,
			EntityID:        payload.EntityID,
			CreatorID:       userID,
			Title:           payload.Title,
			Description:     "",
			Thumbnail:       "",
			Tags:            []string{},
			State:           models.LiveCreated,
			IsPublic:        isPublic,
			StreamKey:       utils.GenerateRandomDigitString(32),
			IngestURL:       app.Config.RTMPIngestURL,
			CreatedAt:       time.Now(),
			ChatEnabled:     true,
			VODPublished:    false,
			Unlisted:        false,
			SlowModeSeconds: 0,
		}

		if payload.Description != nil {
			stream.Description = *payload.Description
		}
		if payload.Thumbnail != nil {
			stream.Thumbnail = *payload.Thumbnail
		}
		if payload.Tags != nil {
			stream.Tags = payload.Tags
		}
		if payload.SlowModeSeconds != nil {
			stream.SlowModeSeconds = *payload.SlowModeSeconds
		}

		if err := app.DB.Insert(r.Context(), vlivesCollection, stream); err != nil {
			log.Printf("CreateStream: insert failed: %v", err)
			writeError(w, "insert failed", http.StatusInternalServerError)
			return
		}

		resp := bson.M{
			"liveid":          stream.LiveID,
			"entityType":      stream.EntityType,
			"entityId":        stream.EntityID,
			"title":           stream.Title,
			"description":     stream.Description,
			"thumbnail":       stream.Thumbnail,
			"tags":            stream.Tags,
			"state":           stream.State,
			"isPublic":        stream.IsPublic,
			"ingestUrl":       stream.IngestURL,
			"createdAt":       stream.CreatedAt,
			"slowModeSeconds": stream.SlowModeSeconds,
			"streamKey":       stream.StreamKey, // exposed only here
		}

		log.Printf("CreateStream: user=%s created liveID=%s", userID, stream.LiveID)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, resp)
	}
}

// ----------------------- MARK READY -----------------------

func MarkReady(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := fetchStream(r.Context(), app, ps.ByName("liveid"), userID, true)
		if err != nil {
			writeError(w, err.Error(), http.StatusForbidden)
			return
		}

		if err := EnforceTransition(stream.State, models.LiveReady); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ok, err := conditionalStateUpdate(r.Context(), app, stream.LiveID, models.LiveCreated,
			bson.M{"state": models.LiveReady, "ready_at": time.Now()})
		if err != nil {
			log.Printf("MarkReady: update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			writeError(w, "state transition failed", http.StatusConflict)
			return
		}

		log.Printf("MarkReady: liveID=%s user=%s", stream.LiveID, userID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ----------------------- SET PRIVACY -----------------------

func SetPrivacy(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := fetchStream(r.Context(), app, ps.ByName("liveid"), userID, true)
		if err != nil {
			writeError(w, err.Error(), http.StatusForbidden)
			return
		}

		var payload struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, "invalid body", http.StatusBadRequest)
			return
		}

		update := bson.M{}
		switch strings.ToUpper(payload.Mode) {
		case "PUBLIC":
			update["is_public"] = true
			update["unlisted"] = false
		case "UNLISTED":
			update["is_public"] = false
			update["unlisted"] = true
		case "PRIVATE":
			update["is_public"] = false
			update["unlisted"] = false
		default:
			writeError(w, "invalid mode", http.StatusBadRequest)
			return
		}

		update["updated_at"] = time.Now()

		err = app.DB.UpdateOne(
			r.Context(),
			vlivesCollection,
			bson.M{"liveid": stream.LiveID},
			bson.M{"$set": update},
		)
		if err != nil {
			log.Printf("SetPrivacy: update failed: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		log.Printf(
			"SetPrivacy: liveID=%s set mode=%s by user=%s",
			stream.LiveID,
			payload.Mode,
			userID,
		)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ----------------------- VIEWER COUNT -----------------------

func GetViewerCount(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		liveID := ps.ByName("liveid")
		if liveID == "" {
			writeError(w, "invalid liveid", http.StatusBadRequest)
			return
		}
		count := getViewerCountCache(r.Context(), app, liveID)
		writeJSON(w, bson.M{"count": count})
	}
}

// ----------------------- TURN SERVERS -----------------------

func GetTURNServers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		writeJSON(w, app.Config.TURNServers)
	}
}
