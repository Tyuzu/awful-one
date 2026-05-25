package farms

import (
	"context"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetFarmDash(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid user",
			})
			return
		}

		var farm models.Farm
		if err := app.DB.FindOne(
			ctx,
			farmsCollection,
			bson.M{"createdBy": userID},
			&farm,
		); err != nil {
			utils.RespondWithJSON(w, http.StatusNotFound, utils.M{
				"success": false,
				"message": "Farm not found",
			})
			return
		}

		var crops []models.Crop
		if err := app.DB.FindMany(
			ctx,
			cropsCollection,
			bson.M{"farmid": farm.FarmID},
			&crops,
		); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to load crops",
			})
			return
		}

		// CRITICAL FIX: Populate farmName in each crop for frontend display
		for i := range crops {
			crops[i].FarmName = farm.Name
		}

		farm.Crops = crops

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"farm":    farm,
		})
	}
}
