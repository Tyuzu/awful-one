package jobs

import (
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// ------------------ READ LIST ------------------
func GetJobsRelatedTOEntity(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		entityType := ps.ByName("entitytype")
		entityID := ps.ByName("entityid")

		var jobs []models.BaitosResponse
		if err := app.DB.FindMany(ctx, baitosCollection, map[string]any{
			"entityType": entityType,
			"entityId":   entityID,
		}, &jobs); err != nil {
			http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
			return
		}

		if jobs == nil {
			jobs = []models.BaitosResponse{}
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
	}
}
