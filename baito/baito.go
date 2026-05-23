package baito

import (
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/* ------------------ DELETE ------------------ */

func DeleteBaito(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		userID := utils.GetUserIDFromRequest(r)
		baitoID := ps.ByName("baitoid")

		filter := bson.M{
			"baitoid": baitoID,
			"ownerid": userID,
		}

		_, err := app.DB.DeleteOne(ctx, BaitoCollection, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				utils.RespondWithError(w, http.StatusForbidden, "Baito not found or unauthorized")
				return
			}
			log.Printf("Delete error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete baito")
			return
		}

		utils.RespondWithJSON(w, http.StatusNoContent, map[string]string{})
	}
}

/* ------------------ APPLY ------------------ */

func ApplyToBaito(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		if err := r.ParseMultipartForm(5 << 20); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data")
			return
		}
		defer r.MultipartForm.RemoveAll()

		pitch := strings.TrimSpace(r.FormValue("pitch"))
		if pitch == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Pitch message required")
			return
		}

		appx := models.BaitoApplication{
			BaitoID:     ps.ByName("baitoid"),
			UserID:      utils.GetUserIDFromRequest(r),
			Username:    utils.GetUsernameFromRequest(r),
			Pitch:       pitch,
			SubmittedAt: time.Now(),
		}

		if err := app.DB.Insert(ctx, BaitoAppCollection, appx); err != nil {
			log.Printf("Insert error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save application")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Application submitted",
		})
	}
}
