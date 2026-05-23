package musicon

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --------------------------- Playlist Handlers ---------------------------

func GetUserPlaylists(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Exclude special likes playlist from normal playlists list
		filter := bson.M{
			"userid": userID,
			"playlistid": bson.M{
				"$ne": "likes_" + userID,
			},
		}

		var playlists []Playlist
		if err := app.DB.FindMany(ctx, playlistsCollection, filter, &playlists); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch playlists")
			return
		}

		respondJSON(w, http.StatusOK, playlists, "Playlists fetched successfully")
	}
}

func CreatePlaylist(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		type Req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		var req Req
		if err := utils.ParseJSON(r, &req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if len(req.Name) == 0 || len(req.Name) > 100 {
			respondError(w, http.StatusBadRequest, "Playlist name must be 1-100 characters")
			return
		}

		now := time.Now()

		newPlaylist := Playlist{
			PlaylistID:    "pl_" + utils.GenerateRandomString(12),
			UserID:        userID,
			Name:          req.Name,
			Description:   req.Description,
			Songs:         []string{},
			Duration:      0,
			IsCompilation: false,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.Insert(ctx, playlistsCollection, newPlaylist); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create playlist")
			return
		}

		respondJSON(w, http.StatusCreated, newPlaylist, "Playlist created successfully")
	}
}

func DeletePlaylist(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		playlistID := ps.ByName("playlistid")

		// Prevent deletion of special likes playlist
		if playlistID == "likes_"+userID {
			respondError(w, http.StatusForbidden, "Cannot delete liked songs playlist")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}

		if _, err := app.DB.DeleteOne(ctx, playlistsCollection, filter); err != nil {
			respondError(w, http.StatusNotFound, "Playlist not found or unauthorized")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"playlist_id": playlistID,
		}, "Playlist deleted successfully")
	}
}

func AddSongToPlaylist(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		playlistID := ps.ByName("playlistid")

		// Prevent manual modification of likes playlist
		if playlistID == "likes_"+userID {
			respondError(w, http.StatusForbidden, "Liked songs playlist cannot be modified directly")
			return
		}

		var body struct {
			SongID string `json:"songid"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if body.SongID == "" {
			respondError(w, http.StatusBadRequest, "Missing song ID")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}

		update := bson.M{
			"$addToSet": bson.M{"songs": body.SongID},
			"$set":      bson.M{"updatedAt": time.Now()},
		}

		if err := app.DB.UpdateOne(ctx, playlistsCollection, filter, update); err != nil {
			respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"playlist_id": playlistID,
			"song_id":     body.SongID,
		}, "Song added to playlist")
	}
}

func RemoveSongFromPlaylist(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		playlistID := ps.ByName("playlistid")
		songID := ps.ByName("songid")

		// Prevent manual modification of likes playlist
		if playlistID == "likes_"+userID {
			respondError(w, http.StatusForbidden, "Liked songs playlist cannot be modified directly")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}

		update := bson.M{
			"$pull": bson.M{"songs": songID},
			"$set":  bson.M{"updatedAt": time.Now()},
		}

		if err := app.DB.UpdateOne(ctx, playlistsCollection, filter, update); err != nil {
			respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"playlist_id": playlistID,
			"song_id":     songID,
		}, "Song removed from playlist")
	}
}

func UpdatePlaylistInfo(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		playlistID := ps.ByName("playlistid")

		// Prevent editing of likes playlist
		if playlistID == "likes_"+userID {
			respondError(w, http.StatusForbidden, "Liked songs playlist cannot be modified")
			return
		}

		type Req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			CoverURL    string `json:"coverUrl"`
		}

		var req Req
		if err := utils.ParseJSON(r, &req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		if len(req.Name) == 0 || len(req.Name) > 100 {
			respondError(w, http.StatusBadRequest, "Playlist name must be 1-100 characters")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}

		update := bson.M{
			"$set": bson.M{
				"name":        req.Name,
				"description": req.Description,
				"coverUrl":    req.CoverURL,
				"updatedAt":   time.Now(),
			},
		}

		if err := app.DB.UpdateOne(ctx, playlistsCollection, filter, update); err != nil {
			respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"playlist_id": playlistID,
		}, "Playlist updated successfully")
	}
}
