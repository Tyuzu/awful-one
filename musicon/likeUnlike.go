package musicon

import (
	"context"
	"fmt"
	"log"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func LikeSong(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		songID := ps.ByName("songid")
		if songID == "" {
			respondError(w, http.StatusBadRequest, "Missing song ID")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		playlistID := "likes_" + userID
		now := time.Now()

		filter := bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}

		update := bson.M{
			"$setOnInsert": bson.M{
				"playlistid":  playlistID,
				"userid":      userID,
				"name":        "Liked Songs",
				"description": "Auto-generated liked songs playlist",
				"songs":       []string{},
				"duration":    0,
				"createdAt":   now,
			},
			"$addToSet": bson.M{
				"songs": songID,
			},
			"$set": bson.M{
				"updatedAt": now,
			},
		}

		err := app.DB.Upsert(ctx, playlistsCollection, filter, update)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to like song")
			return
		}

		respondJSON(w, http.StatusOK, bson.M{
			"song_id": songID,
			"liked":   true,
		}, "Song liked successfully")
	}
}
func UnlikeSong(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		songID := ps.ByName("songid")
		if songID == "" {
			respondError(w, http.StatusBadRequest, "Missing song ID")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		playlistID := "likes_" + userID
		now := time.Now()

		filter := bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}

		update := bson.M{
			"$pull": bson.M{
				"songs": songID,
			},
			"$set": bson.M{
				"updatedAt": now,
			},
		}

		err := app.DB.UpdateOne(ctx, playlistsCollection, filter, update)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to unlike song")
			return
		}

		respondJSON(w, http.StatusOK, bson.M{
			"song_id": songID,
			"liked":   false,
		}, "Song unliked successfully")
	}
}

// --------------------------- User Likes ---------------------------

func GetUserLikes(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
			return
		}

		playlistID := fmt.Sprintf("likes_%s", userID)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var playlist Playlist
		err := app.DB.FindOne(ctx, playlistsCollection, bson.M{
			"playlistid": playlistID,
			"userid":     userID,
		}, &playlist)

		if err != nil || len(playlist.Songs) == 0 {
			respondJSON(w, http.StatusOK, []Song{}, "No liked songs found")
			return
		}

		songs, err := fetchSongsByIDs(ctx, playlist.Songs, app)
		if err != nil {
			log.Printf("GetUserLikes fetch error: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to fetch liked songs")
			return
		}

		respondJSON(w, http.StatusOK, songs, "Liked songs fetched successfully")
	}
}
