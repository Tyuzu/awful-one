package merch

import (
	"context"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		entityType := ps.ByName("entityType")
		eventID := ps.ByName("eventid")
		merchID := ps.ByName("merchid")

		if !validateEntityType(entityType) {
			respond(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   "invalid entity type",
			})
			return
		}

		var merch models.Merch
		err := app.DB.FindOne(
			r.Context(),
			merchCollection,
			bson.M{
				"entity_type": entityType,
				"entity_id":   eventID,
				"merchid":     merchID,
				"deletedAt":   bson.M{"$exists": false},
			},
			&merch,
		)
		if err != nil {
			respond(w, http.StatusNotFound, map[string]any{
				"success": false,
				"error":   "merch not found",
			})
			return
		}

		respond(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    merch,
		})
	}
}

func GetMerchs(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		entityType := ps.ByName("entityType")
		eventID := ps.ByName("eventid")

		if !validateEntityType(entityType) {
			respond(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   "invalid entity type",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var list []models.Merch
		err := app.DB.FindMany(
			ctx,
			merchCollection,
			bson.M{
				"entity_type": entityType,
				"entity_id":   eventID,
				"deletedAt":   bson.M{"$exists": false},
			},
			&list,
		)
		if err != nil {
			respond(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"error":   "failed to fetch merch",
			})
			return
		}

		if list == nil {
			list = []models.Merch{}
		}

		respond(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    list,
		})
	}
}

func GetMerchPage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		merchID := ps.ByName("entityType") // route constraint

		var merch models.Merch
		err := app.DB.FindOne(
			r.Context(),
			merchCollection,
			bson.M{
				"merchid":   merchID,
				"deletedAt": bson.M{"$exists": false},
			},
			&merch,
		)
		if err != nil {
			respond(w, http.StatusNotFound, map[string]any{
				"success": false,
				"error":   "merch not found",
			})
			return
		}

		respond(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    merch,
		})
	}
}
