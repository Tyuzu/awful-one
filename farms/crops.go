package farms

import (
	"context"
	"net/http"
	"strings"
	"time"

	"naevis/dels"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func AddCrop(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		farmID := ps.ByName("id")
		userID := utils.GetUserIDFromRequest(r)

		if farmID == "" || userID == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid farm or user",
			})
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid form data",
			})
			return
		}

		if r.FormValue("name") == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Crop name is required",
			})
			return
		}

		crop := parseCropForm(r)
		crop.FarmID = farmID
		crop.CreatedBy = userID

		if err := app.DB.InsertOne(ctx, cropsCollection, crop); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to create crop",
			})
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"cropId":  crop.CropId,
		})
	}
}

func EditCrop(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cropID := ps.ByName("cropid")
		if cropID == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid crop ID",
			})
			return
		}

		if utils.GetUserIDFromRequest(r) == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid user",
			})
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid form data",
			})
			return
		}

		update := bson.M{
			"updatedAt": time.Now(),
		}

		if v := r.FormValue("name"); v != "" {
			update["name"] = v
		}
		if v := r.FormValue("unit"); v != "" {
			update["unit"] = v
		}
		if v := r.FormValue("price"); v != "" {
			update["price"] = utils.ParseFloat(v)
		}
		if v := r.FormValue("quantity"); v != "" {
			update["quantity"] = utils.ParseInt(v)
		}
		if v := r.FormValue("notes"); v != "" {
			update["notes"] = v
		}
		if v := r.FormValue("category"); v != "" {
			update["category"] = v
		}
		if v := r.FormValue("featured"); v != "" {
			update["featured"] = v == "true"
		}
		if v := r.FormValue("outOfStock"); v != "" {
			update["outOfStock"] = v == "true"
		}

		if d := utils.ParseDate(r.FormValue("harvestDate")); d != nil {
			update["harvestDate"] = d
		}
		if d := utils.ParseDate(r.FormValue("expiryDate")); d != nil {
			update["expiryDate"] = d
		}

		if len(update) == 1 {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "No fields to update",
			})
			return
		}

		if err := app.DB.UpdateOne(
			ctx,
			cropsCollection,
			bson.M{"cropid": cropID},
			bson.M{"$set": update},
		); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Update failed",
			})
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{"success": true})
	}
}

func parseCropForm(r *http.Request) models.Crop {
	name := r.FormValue("name")
	catalogueID := strings.ToLower(strings.ReplaceAll(name, " ", "_"))

	crop := models.Crop{
		CropId:      utils.GenerateRandomString(13),
		Name:        name,
		Price:       utils.ParseFloat(r.FormValue("price")),
		Quantity:    utils.ParseInt(r.FormValue("quantity")),
		Unit:        r.FormValue("unit"),
		Notes:       r.FormValue("notes"),
		Category:    r.FormValue("category"),
		Featured:    r.FormValue("featured") == "true",
		OutOfStock:  r.FormValue("outOfStock") == "true",
		CatalogueId: catalogueID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if d := utils.ParseDate(r.FormValue("harvestDate")); d != nil {
		crop.HarvestDate = d
	}
	if d := utils.ParseDate(r.FormValue("expiryDate")); d != nil {
		crop.ExpiryDate = d
	}

	return crop
}

func DeleteCrop(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		dels.DeleteCrop(app)
	}
}
