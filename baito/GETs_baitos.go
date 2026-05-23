package baito

import (
	"log"
	"net/http"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/* -------------------- Helpers -------------------- */

func respondBaitos(w http.ResponseWriter, baitos []models.BaitosResponse) {
	if baitos == nil {
		baitos = []models.BaitosResponse{}
	}
	utils.RespondWithJSON(w, http.StatusOK, baitos)
}

/* -------------------- Handlers -------------------- */

func GetLatestBaitos(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		var baitos []models.BaitosResponse
		err := app.DB.FindManyWithOptions(
			ctx,
			BaitoCollection,
			bson.M{},
			db.FindManyOptions{
				Limit: 20,
				Sort:  bson.D{{Key: "createdAt", Value: -1}},
			},
			&baitos,
		)
		if err != nil {
			log.Printf("DB error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		respondBaitos(w, baitos)
	}
}

func GetRelatedBaitos(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		category := r.URL.Query().Get("category")
		exclude := r.URL.Query().Get("exclude")

		filter := map[string]any{}
		if category != "" {
			filter["category"] = category
		}
		if exclude != "" {
			filter["baitoid_ne"] = exclude
		}

		var baitos []models.BaitosResponse
		err := app.DB.FindManyWithOptions(
			ctx,
			BaitoCollection,
			filter,
			db.FindManyOptions{
				Limit: 10,
				Sort:  bson.D{{Key: "createdAt", Value: -1}},
			},
			&baitos,
		)
		if err != nil {
			log.Printf("DB error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		// fallback if none found
		if len(baitos) == 0 {
			fallback := map[string]any{}
			if exclude != "" {
				fallback["baitoid_ne"] = exclude
			}

			_ = app.DB.FindManyWithOptions(
				ctx,
				BaitoCollection,
				fallback,
				db.FindManyOptions{
					Limit: 10,
					Sort:  bson.D{{Key: "createdAt", Value: -1}},
				},
				&baitos,
			)
		}

		respondBaitos(w, baitos)
	}
}

func GetBaitoByID(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		id := ps.ByName("baitoid")

		var b models.Baito
		err := app.DB.FindOne(ctx, BaitoCollection, bson.M{"baitoid": id}, &b)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				utils.RespondWithError(w, http.StatusNotFound, "Not found")
			} else {
				log.Printf("DB error: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
			}
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, b)
	}
}

func GetMyBaitos(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		userID := utils.GetUserIDFromRequest(r)

		var baitos []models.BaitosResponse
		err := app.DB.FindManyWithOptions(
			ctx,
			BaitoCollection,
			bson.M{"ownerId": userID},
			db.FindManyOptions{
				Sort: bson.D{{Key: "createdAt", Value: -1}},
			},
			&baitos,
		)
		if err != nil {
			log.Printf("DB error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		respondBaitos(w, baitos)
	}
}

func GetBaitoApplicants(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		baitoID := ps.ByName("baitoid")

		var results []bson.M
		err := app.DB.FindMany(
			ctx,
			BaitoAppCollection,
			bson.M{"baitoid": baitoID},
			&results,
		)
		if err != nil {
			log.Printf("DB error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, results)
	}
}

func GetMyApplications(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		userID := utils.GetUserIDFromRequest(r)

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"userid": userID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         BaitoCollection,
				"localField":   "baitoid",
				"foreignField": "baitoid",
				"as":           "job",
			}}},
			{{Key: "$unwind", Value: "$job"}},
			{{Key: "$project", Value: bson.M{
				"id":          "$_id",
				"pitch":       1,
				"submittedAt": 1,
				"jobId":       "$job.baitoid",
				"title":       "$job.title",
				"location":    "$job.location",
				"wage":        "$job.wage",
			}}},
		}

		var results []bson.M
		err := app.DB.Aggregate(
			ctx,
			BaitoAppCollection,
			pipeline,
			&results,
		)
		if err != nil {
			log.Printf("Aggregate error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch applications")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, results)
	}
}
