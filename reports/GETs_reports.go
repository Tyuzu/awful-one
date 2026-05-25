package reports

import (
	"net/http"
	"strconv"

	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetAppeals(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		q := r.URL.Query()

		status := stringTrim(q.Get("status"))
		if status == "" {
			status = "pending"
		}

		filter := bson.M{"status": status}

		limit := int64(20)
		offset := int64(0)

		if l := stringTrim(q.Get("limit")); l != "" {
			if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 {
				limit = v
			}
		}
		if o := stringTrim(q.Get("offset")); o != "" {
			if v, err := strconv.ParseInt(o, 10, 64); err == nil && v >= 0 {
				offset = v
			}
		}

		var appeals []bson.M
		if err := app.DB.FindMany(ctx, appealsCollection, filter, &appeals); err != nil {
			writeError(w, "Failed to fetch appeals", http.StatusInternalServerError)
			return
		}

		utils.SortAndSlice(
			&appeals,
			bson.D{{Key: "createdAt", Value: -1}},
			offset,
			limit,
		)

		if appeals == nil {
			appeals = []bson.M{}
		}

		writeJSON(w, appeals, http.StatusOK)
	}
}
