package artists

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"naevis/config/mqevent"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	_ "net/http/pprof"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func PostNewSong(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		artistID := ps.ByName("id")

		var payload struct {
			Title       string `json:"title"`
			Genre       string `json:"genre"`
			Duration    string `json:"duration"`
			Description string `json:"description"`
			Audio       string `json:"audio"`
			Poster      string `json:"poster"`
			AudioExtn   string `json:"audioextn"`
			PosterExtn  string `json:"posterextn"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if payload.Title == "" || payload.Genre == "" || payload.Duration == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing required fields: title, genre, duration")
			return
		}

		newSong := models.ArtistSong{
			SongID:      utils.GenerateRandomString(12),
			ArtistID:    artistID,
			Title:       payload.Title,
			Genre:       payload.Genre,
			Duration:    payload.Duration,
			Description: payload.Description,
			AudioURL:    payload.Audio,
			Poster:      payload.Poster,
			Published:   true,
			Plays:       0,
			UploadedAt:  time.Now(),
			AudioExtn:   payload.AudioExtn,
			PosterExtn:  payload.PosterExtn,
		}

		if err := app.DB.Insert(ctx, SongsCollection, newSong); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save song")
			return
		}

		/* -------- Publish SongCreated Event -------- */
		songPayload := mqevent.SongCreatedPayload{
			SongID:     newSong.SongID,
			ArtistID:   artistID,
			SongTitle:  newSong.Title,
			OccurredAt: time.Now(),
		}

		songBytes, err := json.Marshal(songPayload)
		if err == nil {
			publishCtx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			_ = app.MQ.Publish(
				publishCtx,
				mqevent.SongCreated,
				songBytes,
			)
		}

		utils.RespondWithJSON(w, http.StatusCreated, newSong)
	}
}
func EditSong(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		artistID := ps.ByName("id")
		songID := ps.ByName("songId")

		if songID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "songId is required")
			return
		}

		var payload struct {
			Title       string `json:"title"`
			Genre       string `json:"genre"`
			Duration    string `json:"duration"`
			Description string `json:"description"`
			Audio       string `json:"audio"`
			Poster      string `json:"poster"`
			AudioExtn   string `json:"audioextn"`
			PosterExtn  string `json:"posterextn"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		updateFields := bson.M{}
		if payload.Title != "" {
			updateFields["title"] = payload.Title
		}
		if payload.Genre != "" {
			updateFields["genre"] = payload.Genre
		}
		if payload.Duration != "" {
			updateFields["duration"] = payload.Duration
		}
		if payload.Description != "" {
			updateFields["description"] = payload.Description
		}
		if payload.Audio != "" {
			updateFields["audioUrl"] = payload.Audio
		}
		if payload.AudioExtn != "" {
			updateFields["audioextn"] = payload.AudioExtn
		}
		if payload.Poster != "" {
			updateFields["poster"] = payload.Poster
		}
		if payload.PosterExtn != "" {
			updateFields["posterextn"] = payload.PosterExtn
		}

		if len(updateFields) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "No fields to update")
			return
		}

		updateFields["updatedAt"] = time.Now()

		filter := bson.M{"songid": songID, "artistid": artistID}
		update := bson.M{"$set": updateFields}

		err := app.DB.Update(ctx, SongsCollection, filter, update)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update song")
			return
		}

		/* -------- Publish SongUpdated Event -------- */
		songUpdatePayload := mqevent.SongUpdatedPayload{
			SongID:     songID,
			ArtistID:   artistID,
			OccurredAt: time.Now(),
		}

		songUpdateBytes, err := json.Marshal(songUpdatePayload)
		if err == nil {
			publishCtx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			_ = app.MQ.Publish(
				publishCtx,
				mqevent.SongUpdated,
				songUpdateBytes,
			)
		}

		utils.RespondWithJSON(w, http.StatusOK, bson.M{"message": "Song updated successfully"})
	}
}
func DeleteSong(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		artistID := ps.ByName("id")
		songID := ps.ByName("songId")

		if songID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "songId is required")
			return
		}

		filter := bson.M{"artistid": artistID, "songid": songID}

		_, err := app.DB.Delete(ctx, SongsCollection, filter)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete song")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, bson.M{"message": "Song deleted successfully"})
	}
}
