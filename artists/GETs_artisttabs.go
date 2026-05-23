package artists

import (
	"naevis/infra"
	"naevis/media"
	"naevis/merch"
	"naevis/models"
	"naevis/utils"
	"net/http"
	_ "net/http/pprof"

	"github.com/julienschmidt/httprouter"
)

func GetArtistsAlbums(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		albums := []models.ArtistAlbum{
			{
				Title:       "Nightfall",
				ReleaseDate: "2023-10-01",
				Description: "A journey through dusk.",
				Published:   true,
			},
			{
				Title:       "Drip",
				ReleaseDate: "2025-11-17",
				Description: "A journey till dawn.",
				Published:   true,
			},
			{
				Title:       "Unreleased Gems",
				ReleaseDate: "2025-01-01",
				Description: "Upcoming exclusives.",
				Published:   false,
			},
		}

		utils.RespondWithJSON(w, http.StatusOK, albums)
	}
}

func GetArtistsPosts(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ps = append(ps,
			httprouter.Param{Key: "entityid", Value: ps.ByName("id")},
			httprouter.Param{Key: "entitytype", Value: "artist"},
		)

		media.GetMedias(app)(w, r, ps)
	}
}

func GetArtistsMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ps = append(ps,
			httprouter.Param{Key: "eventid", Value: ps.ByName("id")},
			httprouter.Param{Key: "entityType", Value: "artist"},
		)

		merch.GetMerchs(app)(w, r, ps)
	}
}

func GetArtistsevents(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		events := []models.ArtistEvent{
			{Title: "Summer Fest 2025", Date: "2025-06-15", Venue: "Sunset Arena", City: "Los Angeles", Country: "USA", TicketURL: "http://localhost:5173/event/4s89t5jt6754djt"},
			{Title: "Berlin Beats", Date: "2025-07-20", Venue: "Techno Temple", City: "Berlin", Country: "Germany"},
		}
		utils.RespondWithJSON(w, http.StatusOK, events)
	}
}
