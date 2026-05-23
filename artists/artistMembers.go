package artists

import (
	"encoding/json"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func AddArtistMember(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		artistID := ps.ByName("id")

		// Ensure artist exists
		var artist models.Artist
		if err := app.DB.FindOne(ctx, ArtistsCollection, bson.M{
			"artistid": artistID,
		}, &artist); err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Artist not found")
			return
		}

		var m models.BandMember
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		m.Name = strings.TrimSpace(m.Name)
		m.Role = strings.TrimSpace(m.Role)
		m.DOB = strings.TrimSpace(m.DOB)
		m.Image = strings.TrimSpace(m.Image)
		m.ReferenceArtist = strings.TrimSpace(m.ReferenceArtist)

		if m.Name == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Member name is required")
			return
		}

		if m.MemberID == "" {
			m.MemberID = utils.GenerateRandomString(12)
		}

		// Prevent duplicate by referenced artist
		if m.ReferenceArtist != "" {
			for _, existing := range artist.Members {
				if existing.ReferenceArtist == m.ReferenceArtist {
					utils.RespondWithError(w, http.StatusConflict, "Referenced artist already added")
					return
				}
			}
		}

		update := bson.M{
			"$push": bson.M{"members": m},
		}

		if err := app.DB.Update(
			ctx,
			ArtistsCollection,
			bson.M{"artistid": artistID},
			update,
		); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to add member")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, m)
	}
}

func UpdateArtistMember(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		artistID := ps.ByName("id")
		memberID := ps.ByName("memberId")

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		updates := bson.M{}

		if v, ok := payload["name"]; ok {
			updates["members.$.name"] = strings.TrimSpace(v)
		}
		if v, ok := payload["role"]; ok {
			updates["members.$.role"] = strings.TrimSpace(v)
		}
		if v, ok := payload["dob"]; ok {
			updates["members.$.dob"] = strings.TrimSpace(v)
		}
		if v, ok := payload["image"]; ok {
			updates["members.$.image"] = strings.TrimSpace(v)
		}

		if len(updates) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "No valid fields to update")
			return
		}

		filter := bson.M{
			"artistid":         artistID,
			"members.memberid": memberID,
		}

		err := app.DB.Update(
			ctx,
			ArtistsCollection,
			filter,
			bson.M{"$set": updates},
		)

		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update member")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, bson.M{
			"message": "Member updated",
		})
	}
}

func DeleteArtistMember(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		artistID := ps.ByName("id")
		memberID := ps.ByName("memberId")

		update := bson.M{
			"$pull": bson.M{
				"members": bson.M{"memberid": memberID},
			},
		}

		if err := app.DB.Update(
			ctx,
			ArtistsCollection,
			bson.M{"artistid": artistID},
			update,
		); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete member")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, bson.M{
			"message": "Member deleted",
		})
	}
}
