package baito

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/* -------------------- Workers -------------------- */

func GetWorkerById(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var worker models.BaitoWorker
		err := app.DB.FindOne(
			ctx,
			BaitoWorkersCollection,
			bson.M{"baitoUserId": ps.ByName("workerId")},
			&worker,
		)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				utils.RespondWithError(w, http.StatusNotFound, "Worker not found")
			} else {
				log.Printf("DB error: %v", err)
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch worker")
			}
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, worker)
	}
}

func GetWorkerSkills(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		/*
			Distinct is intentionally implemented via aggregation
			to keep the Database interface Mongo-agnostic
		*/
		pipeline := mongo.Pipeline{
			{{Key: "$unwind", Value: "$preferredRoles"}},
			{{Key: "$group", Value: bson.M{"_id": "$preferredRoles"}}},
			{{Key: "$project", Value: bson.M{"_id": 0, "skill": "$_id"}}},
		}

		var results []bson.M
		err := app.DB.Aggregate(ctx, BaitoWorkersCollection, pipeline, &results)
		if err != nil {
			log.Printf("Aggregate error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch skills")
			return
		}

		skills := make([]string, 0, len(results))
		for _, r := range results {
			if s, ok := r["skill"].(string); ok && s != "" {
				skills = append(skills, s)
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, skills)
	}
}

// GetWorkers returns a list of workers with optional search and skill filtering.
func GetWorkers(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		search := strings.TrimSpace(r.URL.Query().Get("search"))
		skill := strings.TrimSpace(r.URL.Query().Get("skill"))

		filter := map[string]any{}

		if search != "" {
			filter["$or"] = []any{
				map[string]any{"name_contains": search},
				map[string]any{"address_contains": search},
				map[string]any{"bio_contains": search},
			}
		}

		if skill != "" {
			filter["preferredRoles"] = skill
		}

		skip, limit := utils.ParsePagination(r, 10, 100)

		opts := db.FindManyOptions{
			Skip:  skip,
			Limit: limit,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var workers []models.BaitoWorkersResponse
		if err := app.DB.FindManyWithOptions(ctx, BaitoWorkersCollection, filter, opts, &workers); err != nil {
			log.Printf("DB error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch workers")
			return
		}

		total, err := app.DB.CountDocuments(ctx, BaitoWorkersCollection, filter)
		if err != nil {
			log.Printf("Count error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch workers")
			return
		}

		if workers == nil {
			workers = []models.BaitoWorkersResponse{}
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"data":  workers,
			"total": total,
		})
	}
}
