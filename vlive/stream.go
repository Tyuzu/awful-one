package vlive

import (
	"encoding/json"
	"log"
	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// ----------------------- GET STREAM -----------------------

func GetStream(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)

		stream, err := fetchStream(
			r.Context(),
			app,
			ps.ByName("liveid"),
			userID,
			false,
		)
		if err != nil {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}

		if !stream.IsPublic && stream.CreatorID != userID {
			if !CheckEntityAccess(
				r.Context(),
				app,
				userID,
				stream.EntityType,
				stream.EntityID,
			) {
				writeError(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if stream.State == models.LiveCancelled {
			writeError(w, "stream cancelled", http.StatusGone)
			return
		}

		stream.ViewerCount =
			getViewerCountCache(r.Context(), app, stream.LiveID)

		if !stream.VODPublished && stream.CreatorID != userID {
			stream.VODURL = ""
		}

		availability := "unknown"
		switch stream.State {
		case models.LiveLive:
			availability = "live"
		case models.LiveReady, models.LiveCreated:
			availability = "scheduled"
		case models.LiveEnded:
			availability = "ended"
		case models.LiveCancelled:
			availability = "cancelled"
		case models.LiveError:
			availability = "error"
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
			"availability":    availability,
			"isPublic":        stream.IsPublic,
			"playbackUrl":     stream.PlaybackURL,
			"vodUrl":          stream.VODURL,
			"chatEnabled":     stream.ChatEnabled,
			"slowModeSeconds": stream.SlowModeSeconds,
			"scheduledAt":     stream.ScheduledAt,
			"readyAt":         stream.ReadyAt,
			"startedAt":       stream.StartedAt,
			"endedAt":         stream.EndedAt,
			"viewerCount":     stream.ViewerCount,
		}

		writeJSON(w, resp)
	}
}

// ----------------------- LIST STREAMS -----------------------

func ListStreams(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		entityType := r.URL.Query().Get("entityType")
		entityID := r.URL.Query().Get("entityId")
		userID := utils.GetUserIDFromRequest(r)

		if entityType == "" || entityID == "" {
			writeError(w, "entityType and entityId required", http.StatusBadRequest)
			return
		}

		// 🔒 entity access check (non-public streams)
		allowed := false
		if userID != "" {
			allowed = CheckEntityAccess(
				r.Context(),
				app,
				userID,
				entityType,
				entityID,
			)
		}

		filter := bson.M{
			"entity_type": entityType,
			"entity_id":   entityID,
			"$or": []bson.M{
				{"is_public": true},
				{"creator_id": userID},
			},
			"$nor": []bson.M{
				{
					"state": bson.M{
						"$in": []string{
							models.LiveCancelled,
							models.LiveError,
						},
					},
					"creator_id": bson.M{"$ne": userID},
				},
			},
		}

		// If user has entity access, allow private streams too
		if allowed {
			filter["$or"] = []bson.M{
				{"is_public": true},
				{"creator_id": userID},
				{"entity_id": entityID},
			}
		}

		opts := db.FindManyOptions{
			Sort: bson.D{
				{Key: "state", Value: 1},
				{Key: "scheduled_at", Value: 1},
				{Key: "started_at", Value: -1},
				{Key: "created_at", Value: -1},
			},
		}

		var streams []models.LiveStream
		err := app.DB.FindManyWithOptions(
			r.Context(),
			vlivesCollection,
			filter,
			opts,
			&streams,
		)
		if err != nil {
			log.Printf("ListStreams: find error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		for i := range streams {
			streams[i].StreamKey = ""
			streams[i].IngestURL = ""
			streams[i].RecordingPath = ""
			streams[i].ViewerCount =
				getViewerCountCache(r.Context(), app, streams[i].LiveID)
		}

		writeJSON(w, streams)
	}
}

// ----------------------- START STREAM -----------------------

func StartStream(app *infra.Deps) httprouter.Handle {
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

		if err := EnforceTransition(stream.State, models.LiveLive); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		playbackURL := strings.TrimRight(app.Config.CDNBaseURL, "/") + "/live/" + stream.LiveID + ".m3u8"
		ok, err := conditionalStateUpdate(r.Context(), app, stream.LiveID, models.LiveReady,
			bson.M{"state": models.LiveLive, "started_at": time.Now(), "playback_url": playbackURL})
		if err != nil {
			log.Printf("StartStream: update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			writeError(w, "state transition failed", http.StatusConflict)
			return
		}

		log.Printf("StartStream: liveID=%s started by user=%s", stream.LiveID, userID)
		w.WriteHeader(http.StatusOK)
	}
}

// ----------------------- STOP STREAM -----------------------

func StopStream(app *infra.Deps) httprouter.Handle {
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

		if err := EnforceTransition(stream.State, models.LiveEnded); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ok, err := conditionalStateUpdate(r.Context(), app, stream.LiveID, models.LiveLive,
			bson.M{"state": models.LiveEnded, "ended_at": time.Now()})
		if err != nil {
			log.Printf("StopStream: update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			writeError(w, "cannot stop stream in current state", http.StatusConflict)
			return
		}

		log.Printf("StopStream: liveID=%s stopped by user=%s", stream.LiveID, userID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ----------------------- UPDATE METADATA -----------------------

func UpdateStreamMetadata(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := fetchStream(
			r.Context(),
			app,
			ps.ByName("liveid"),
			userID,
			true,
		)
		if err != nil {
			writeError(w, err.Error(), http.StatusForbidden)
			return
		}

		if err := disallowWhileLive(stream); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, "invalid body", http.StatusBadRequest)
			return
		}

		update, err := validateMetadata(payload)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(update) == 0 {
			writeError(w, "nothing to update", http.StatusBadRequest)
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
			log.Printf(
				"UpdateStreamMetadata: update failed liveID=%s err=%v",
				stream.LiveID,
				err,
			)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		log.Printf(
			"UpdateStreamMetadata: liveID=%s updated fields=%v",
			stream.LiveID,
			update,
		)
		w.WriteHeader(http.StatusNoContent)
	}
}
