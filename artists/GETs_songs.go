package artists

import (
	"context"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// GetArtistsSongs returns all published songs for an artist.
// If no songs exist, returns an empty array.
func GetArtistsSongs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		artistID := ps.ByName("id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{"artistid": artistID, "published": true}

		var songs []models.ArtistSong
		err := app.DB.FindMany(ctx, SongsCollection, filter, &songs)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch songs")
			return
		}

		// Ensure we always return an array, not null
		if songs == nil {
			songs = []models.ArtistSong{}
		}

		utils.RespondWithJSON(w, http.StatusOK, songs)
	}
}
