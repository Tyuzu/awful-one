package artists

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"naevis/dels"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func CreateArtist(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		artist, _, _, err := parseArtistFormData(r, nil)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		artist.ArtistID = utils.GenerateRandomString(12)
		artist.EventIDs = []string{}

		if err := app.DB.Insert(ctx, ArtistsCollection, artist); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create artist")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, artist)
	}
}

func UpdateArtist(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		idParam := ps.ByName("id")

		if err := r.ParseMultipartForm(20 << 20); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		var existing models.Artist
		if err := app.DB.FindOne(ctx, ArtistsCollection, bson.M{"artistid": idParam}, &existing); err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Artist not found")
			return
		}

		updated, updateData, filesToDelete, err := parseArtistFormData(r, &existing)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = updated

		if len(updateData) == 0 {
			utils.RespondWithJSON(w, http.StatusOK, bson.M{"message": "No changes detected"})
			return
		}

		err = app.DB.Update(ctx, ArtistsCollection, bson.M{"artistid": idParam}, bson.M{"$set": updateData})
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update artist")
			return
		}

		// Cleanup old images only after DB update succeeds
		for _, path := range filesToDelete {
			_ = os.Remove(path)
		}

		utils.RespondWithJSON(w, http.StatusOK, bson.M{"message": "Artist updated"})
	}
}
func parseArtistFormData(r *http.Request, existing *models.Artist) (models.Artist, bson.M, []string, error) {
	var artist models.Artist
	updateData := bson.M{}
	filesToDelete := []string{}

	// Preserve IDs
	if existing != nil {
		artist.ArtistID = existing.ArtistID
		artist.EventIDs = existing.EventIDs
	}

	assignField := func(key string, target *string, existingVal string) {
		if val := r.FormValue(key); val != "" {
			*target = val
			updateData[key] = val
		} else {
			*target = existingVal
		}
	}

	assignField("name", &artist.Name, existingValue(existing, "Name"))
	assignField("bio", &artist.Bio, existingValue(existing, "Bio"))
	assignField("category", &artist.Category, existingValue(existing, "Category"))
	assignField("dob", &artist.DOB, existingValue(existing, "DOB"))
	assignField("place", &artist.Place, existingValue(existing, "Place"))
	assignField("country", &artist.Country, existingValue(existing, "Country"))

	// Creator ID
	artist.CreatorID = utils.GetUserIDFromRequest(r)
	if artist.CreatorID != "" {
		updateData["creatorid"] = artist.CreatorID
	} else if existing != nil {
		artist.CreatorID = existing.CreatorID
	}

	// Genres
	if val := r.FormValue("genres"); val != "" {
		var genres []string
		for _, g := range strings.Split(val, ",") {
			if g = strings.TrimSpace(g); g != "" {
				genres = append(genres, g)
			}
		}
		artist.Genres = genres
		updateData["genres"] = genres
	} else if existing != nil {
		artist.Genres = existing.Genres
	}

	// Socials
	if val := r.FormValue("socials"); val != "" {
		var socials map[string]string
		if err := json.Unmarshal([]byte(val), &socials); err == nil {
			artist.Socials = socials
			updateData["socials"] = socials
		} else {
			artist.Socials = map[string]string{"raw": val}
			updateData["socials"] = artist.Socials
		}
	} else if existing != nil {
		artist.Socials = existing.Socials
	}

	// Preserve members (not updated here)
	if existing != nil {
		artist.Members = existing.Members
	}

	return artist, updateData, filesToDelete, nil
}

func DeleteArtistByID(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		dels.DeleteArtistByID(app)
		// ctx := r.Context()
		// artistID := ps.ByName("id")

		// if artistID == "" {
		// 	utils.RespondWithError(w, http.StatusBadRequest, "artistID is required")
		// 	return
		// }

		// filter := bson.M{"artistid": artistID}
		// update := bson.M{"$set": bson.M{"deleted": true}}

		// _, err := app.DB.ArtistsCollection.UpdateOne(ctx, filter, update)
		// if err != nil {
		// 	utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete artist")
		// 	return
		// }

		// go mq.Emit(ctx, "artist-deleted", models.Index{
		// 	EntityType: "artist", EntityId: artistID, Method: "DELETE",
		// })

		// utils.RespondWithJSON(w, http.StatusOK, bson.M{"message": "Artist deleted successfully"})
	}
}
func existingValue(existing *models.Artist, field string) string {
	if existing == nil {
		return ""
	}
	switch field {
	case "Name":
		return existing.Name
	case "Bio":
		return existing.Bio
	case "Category":
		return existing.Category
	case "DOB":
		return existing.DOB
	case "Place":
		return existing.Place
	case "Country":
		return existing.Country
	case "Banner":
		return existing.Banner
	case "Photo":
		return existing.Photo
	default:
		return ""
	}
}
