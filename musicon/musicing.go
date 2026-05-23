package musicon

import (
	"context"
	"fmt"
	"log"
	"naevis/infra"
	"naevis/infra/db"
	"naevis/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --------------------------- Helpers ---------------------------

// fetchSongsByIDs retrieves published songs by IDs
func fetchSongsByIDs(ctx context.Context, ids []string, app *infra.Deps) ([]Song, error) {
	if len(ids) == 0 {
		return []Song{}, nil
	}

	filter := bson.M{
		"songid":    bson.M{"$in": ids},
		"published": true,
	}

	var songs []Song
	if err := app.DB.FindMany(ctx, songsCollection, filter, &songs); err != nil {
		return nil, err
	}

	return songs, nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}, message string) {
	utils.RespondWithJSON(w, status, map[string]interface{}{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	utils.RespondWithJSON(w, status, map[string]interface{}{
		"success": false,
		"data":    nil,
		"message": message,
	})
}

func getPaginationParams(r *http.Request) (limit int, page int) {
	limit = 20
	page = 1

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	return
}

// --------------------------- Albums & Songs ---------------------------

func GetAlbums(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var albums []Album
		if err := app.DB.FindMany(ctx, albumsCollection, bson.M{"published": true}, &albums); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch albums")
			return
		}

		respondJSON(w, http.StatusOK, albums, "Albums fetched successfully")
	}
}

func GetAlbumSongs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		albumID := ps.ByName("albumid")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var album Album
		if err := app.DB.FindOne(ctx, albumsCollection, bson.M{"albumid": albumID}, &album); err != nil {
			respondJSON(w, http.StatusOK, []Song{}, "No songs found for album")
			return
		}

		songs, err := fetchSongsByIDs(ctx, album.Songs, app)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch songs")
			return
		}

		respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for album %s fetched", albumID))
	}
}

func GetPlaylistSongs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		playlistID := ps.ByName("playlistid")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var playlist Playlist
		if err := app.DB.FindOne(ctx, playlistsCollection, bson.M{"playlistid": playlistID}, &playlist); err != nil {
			respondJSON(w, http.StatusOK, []Song{}, "Playlist not found")
			return
		}

		songs, err := fetchSongsByIDs(ctx, playlist.Songs, app)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch songs")
			return
		}

		respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for playlist %s fetched", playlistID))
	}
}

// --------------------------- Artist Songs ---------------------------
func GetArtistsSongs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		artistID := ps.ByName("artistid")
		if artistID == "" {
			respondError(w, http.StatusBadRequest, "Missing artist ID")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		limit, page := getPaginationParams(r)
		skip := (page - 1) * limit

		filter := bson.M{
			"artistid":  artistID,
			"published": true,
		}

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort: bson.D{{
				Key: "uploadedAt", Value: -1,
			}},
		}

		var songs []Song
		if err := app.DB.FindManyWithOptions(ctx, songsCollection, filter, opts, &songs); err != nil {
			log.Printf("GetArtistsSongs error: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to fetch artist songs")
			return
		}

		respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for artist %s fetched", artistID))
	}
}
