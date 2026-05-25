package moderator

import (
	"encoding/json"
	"net/http"

	"naevis/infra"
	"naevis/models"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetReports(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		filter := bson.M{
			"status": bson.M{
				"$nin": []string{"resolved", "rejected"},
			},
		}

		var reports []models.Report
		err := app.DB.FindMany(ctx, reportsCollection, filter, &reports)
		if err != nil {
			http.Error(w, `{"error":"Failed to fetch reports"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reports)
	}
}
