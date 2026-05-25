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

// Artist Events
func GetArtistEvents(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := bson.M{"artistid": ps.ByName("id")}
		var artistevents []models.ArtistEvent
		err := app.DB.FindMany(ctx, ArtistEventsCollection, filter, &artistevents)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch artist events")
			return
		}

		if len(artistevents) == 0 {
			artistevents = []models.ArtistEvent{}
		}

		utils.RespondWithJSON(w, http.StatusOK, artistevents)
	}
}
func GetArtistByID(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		artistId := ps.ByName("id")
		var artist models.Artist

		// Fetch artist info
		if err := app.DB.FindOne(ctx, ArtistsCollection, bson.M{"artistid": artistId}, &artist); err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Artist not found")
			return
		}

		// Default: not subscribed
		isSubscribed := false

		// Get current logged-in user ID
		currentUserID := utils.GetUserIDFromRequest(r)
		if currentUserID != "" {
			// Check if the user has subscribed to this artist
			var subscribers []bson.M
			err := app.DB.FindMany(ctx, SubscribersCollection, bson.M{
				"userid": currentUserID,
				"subscribed": bson.M{
					"$in": []string{artistId},
				},
			}, &subscribers)
			if err == nil && len(subscribers) > 0 {
				isSubscribed = true
			}
		}

		// Response struct: embed artist + subscription info
		resp := struct {
			models.Artist
			IsSubscribed bool `json:"isSubscribed"`
		}{
			Artist:       artist,
			IsSubscribed: isSubscribed,
		}

		utils.RespondWithJSON(w, http.StatusOK, resp)
	}
}
func GetArtistsByEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		eventID := ps.ByName("eventid")

		var artists []models.Artist
		err := app.DB.FindMany(ctx, ArtistsCollection, bson.M{"events": eventID}, &artists)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error fetching artists")
			return
		}

		if len(artists) == 0 {
			artists = []models.Artist{}
		}

		utils.RespondWithJSON(w, http.StatusOK, artists)
	}
}

// All Artists
func GetAllArtists(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var artists []models.Artist
		err := app.DB.FindMany(ctx, ArtistsCollection, bson.M{}, &artists)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error fetching artists")
			return
		}

		if len(artists) == 0 {
			artists = []models.Artist{}
		}

		utils.RespondWithJSON(w, http.StatusOK, artists)
	}
}
