package recipes

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --- Get single recipe ---

func GetRecipe(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		id := ps.ByName("id")

		var recipe models.Recipe
		if err := app.DB.FindOne(ctx, recipeCollection, map[string]any{"recipeid": id}, &recipe); err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}

		normalizeRecipeSlices(&recipe)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(recipe)
	}
}

// --- List Recipes ---

func GetRecipes(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		filter := map[string]any{}

		if search := r.URL.Query().Get("search"); search != "" {
			filter["$or"] = []any{
				utils.RegexFilter("title", search),
				utils.RegexFilter("description", search),
			}
		}

		if ing := r.URL.Query().Get("ingredient"); ing != "" {
			filter["ingredients.name"] = map[string]any{
				"$regex":   regexp.QuoteMeta(ing),
				"$options": "i",
			}
		}

		if tags := r.URL.Query().Get("tags"); tags != "" {
			filter["tags"] = map[string]any{"$all": strings.Split(tags, ",")}
		}

		skip, limit := utils.ParsePagination(r, 10, 100)
		sort := utils.ParseSort(
			r.URL.Query().Get("sort"),
			bson.D{{Key: "createdAt", Value: -1}},
			map[string]bson.D{
				"newest":   {{Key: "createdAt", Value: -1}},
				"oldest":   {{Key: "createdAt", Value: 1}},
				"views":    {{Key: "views", Value: -1}},
				"prepTime": {{Key: "prepTime", Value: 1}},
			},
		)

		opts := db.FindManyOptions{
			Skip:  skip,
			Limit: limit,
			Sort:  sort,
		}

		var recipes []models.Recipe
		if err := app.DB.FindManyWithOptions(ctx, recipeCollection, filter, opts, &recipes); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch recipes")
			return
		}

		for i := range recipes {
			normalizeRecipeSlices(&recipes[i])
		}

		totalCount, err := app.DB.CountDocuments(ctx, recipeCollection, filter)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to count recipes")
			return
		}

		hasMore := (skip + limit) < int(totalCount)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"recipes": recipes,
			"hasMore": hasMore,
		})
	}
}

// --- Tags ---

type recipeTagAgg struct {
	Tags []string `bson:"tags"`
}

func GetRecipeTags(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		pipeline := []any{
			map[string]any{"$unwind": "$tags"},
			map[string]any{"$group": map[string]any{
				"_id":  nil,
				"tags": map[string]any{"$addToSet": "$tags"},
			}},
		}

		var result []recipeTagAgg
		if err := app.DB.Aggregate(ctx, recipeCollection, pipeline, &result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tags := []string{}
		if len(result) > 0 && result[0].Tags != nil {
			tags = result[0].Tags
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tags)
	}
}
