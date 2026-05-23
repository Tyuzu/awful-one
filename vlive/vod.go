package vlive

import (
	"log"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ----------------------- VOD / INGEST -----------------------

func StartIngest(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		streamKey := r.URL.Query().Get("key")
		if streamKey == "" {
			writeError(w, "missing key", http.StatusBadRequest)
			return
		}

		stream, err := getStreamByKey(r.Context(), app, streamKey)
		if err != nil {
			writeError(w, "invalid stream key", http.StatusUnauthorized)
			return
		}

		allowedFrom := map[string]bool{
			models.LiveReady:   true,
			models.LiveCreated: true,
		}
		if !allowedFrom[stream.State] {
			writeError(w, "stream not ready for ingest", http.StatusConflict)
			return
		}

		playback := strings.TrimRight(app.Config.CDNBaseURL, "/") +
			"/live/" + stream.LiveID + ".m3u8"

		ok, err := conditionalStateUpdate(
			r.Context(),
			app,
			stream.LiveID,
			stream.State,
			bson.M{
				"state":        models.LiveLive,
				"started_at":   time.Now(),
				"playback_url": playback,
			},
		)
		if err != nil || !ok {
			log.Printf("StartIngest: transition error: %v ok=%v", err, ok)
			writeError(w, "failed to start ingest", http.StatusInternalServerError)
			return
		}

		writeJSON(w, bson.M{"status": "ok", "playbackUrl": playback})
	}
}

func RecordingComplete(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		streamKey := r.URL.Query().Get("key")
		file := r.URL.Query().Get("path")
		if streamKey == "" || file == "" {
			writeError(w, "missing key or path", http.StatusBadRequest)
			return
		}

		stream, err := getStreamByKey(r.Context(), app, streamKey)
		if err != nil {
			writeError(w, "invalid stream key", http.StatusUnauthorized)
			return
		}

		vodURL := strings.TrimRight(app.Config.CDNBaseURL, "/") +
			"/vod/" + filepath.Base(file)

		err = app.DB.UpdateOne(
			r.Context(),
			vlivesCollection,
			bson.M{"liveid": stream.LiveID},
			bson.M{
				"$set": bson.M{
					"recording_path": file,
					"vod_url":        vodURL,
					"state":          models.LiveEnded,
					"ended_at":       time.Now(),
				},
			},
		)
		if err != nil {
			log.Printf("RecordingComplete: update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, bson.M{"vodUrl": vodURL})
	}
}

func PublishVOD(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := primitive.ObjectIDFromHex(ps.ByName("liveid"))
		if err != nil {
			writeError(w, "invalid liveid", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := getStreamByID(r.Context(), app, id)
		if err != nil {
			writeError(w, "stream not found", http.StatusNotFound)
			return
		}

		if !isOwner(userID, stream) {
			writeError(w, "forbidden", http.StatusForbidden)
			return
		}

		if stream.VODURL == "" {
			writeError(w, "no vod available", http.StatusBadRequest)
			return
		}

		err = app.DB.UpdateOne(
			r.Context(),
			vlivesCollection,
			bson.M{"_id": id},
			bson.M{"$set": bson.M{"vod_published": true}},
		)
		if err != nil {
			log.Printf("PublishVOD: update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DeleteVOD(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := primitive.ObjectIDFromHex(ps.ByName("liveid"))
		if err != nil {
			writeError(w, "invalid liveid", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := getStreamByID(r.Context(), app, id)
		if err != nil {
			writeError(w, "stream not found", http.StatusNotFound)
			return
		}

		if !isOwner(userID, stream) {
			writeError(w, "forbidden", http.StatusForbidden)
			return
		}

		err = app.DB.UpdateOne(
			r.Context(),
			vlivesCollection,
			bson.M{"_id": id},
			bson.M{
				"$unset": bson.M{
					"vod_url":        "",
					"recording_path": "",
				},
				"$set": bson.M{
					"vod_published": false,
				},
			},
		)
		if err != nil {
			log.Printf("DeleteVOD: update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func GetVOD(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := primitive.ObjectIDFromHex(ps.ByName("liveid"))
		if err != nil {
			writeError(w, "invalid id", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)

		stream, err := getStreamByID(r.Context(), app, id)
		if err != nil || stream.VODURL == "" {
			writeError(w, "not found", http.StatusNotFound)
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

		if !stream.VODPublished && stream.CreatorID != userID {
			writeError(w, "vod not published", http.StatusNotFound)
			return
		}

		writeJSON(w, bson.M{"vodUrl": stream.VODURL})
	}
}
