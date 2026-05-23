package recipes

import (
	"context"
	"encoding/json"
	"naevis/dels"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --- Helpers ---

func splitCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func getSafe(arr []string, index int) string {
	if index < len(arr) {
		return arr[index]
	}
	return ""
}

func normalizeRecipeSlices(r *models.Recipe) {
	if r.Dietary == nil {
		r.Dietary = []string{}
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	if r.Steps == nil {
		r.Steps = []string{}
	}
	if r.Images == nil {
		r.Images = []string{}
	}
	if r.Ingredients == nil {
		r.Ingredients = []models.Ingredient{}
	}

	for i := range r.Ingredients {
		if r.Ingredients[i].Alternatives == nil {
			r.Ingredients[i].Alternatives = []models.IngredientAlternative{}
		}
	}
}

// --- Create Recipe ---

func CreateRecipe(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)

		var servings int
		if v := r.FormValue("servings"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				servings = parsed
			}
		}

		names := r.MultipartForm.Value["ingredientName[]"]
		quantities := r.MultipartForm.Value["ingredientQuantity[]"]
		units := r.MultipartForm.Value["ingredientUnit[]"]
		rawAlts := r.MultipartForm.Value["ingredientAlternatives[]"]

		ingredients := make([]models.Ingredient, 0, len(names))

		for i := range names {
			if names[i] == "" || i >= len(quantities) || i >= len(units) {
				continue
			}

			qty, err := strconv.ParseFloat(quantities[i], 64)
			if err != nil {
				continue
			}

			ingredient := models.Ingredient{
				Name:         names[i],
				Quantity:     qty,
				Unit:         units[i],
				Alternatives: []models.IngredientAlternative{},
			}

			if i < len(rawAlts) && rawAlts[i] != "" {
				for _, alt := range strings.Split(rawAlts[i], ",") {
					parts := strings.Split(alt, "|")
					if len(parts) == 3 {
						ingredient.Alternatives = append(
							ingredient.Alternatives,
							models.IngredientAlternative{
								Name:   parts[0],
								ItemID: parts[1],
								Type:   parts[2],
							},
						)
					}
				}
			}

			ingredients = append(ingredients, ingredient)
		}

		recipe := models.Recipe{
			RecipeId:    utils.GenerateRandomString(12),
			UserID:      userID,
			Title:       r.FormValue("title"),
			Description: r.FormValue("description"),
			CookTime:    r.FormValue("cookTime"),
			Cuisine:     r.FormValue("cuisine"),
			Dietary:     splitCSV(r.FormValue("dietary")),
			PortionSize: r.FormValue("portionSize"),
			Season:      r.FormValue("season"),
			Tags:        splitCSV(r.FormValue("tags")),
			Steps:       splitLines(r.FormValue("steps")),
			Ingredients: ingredients,
			Difficulty:  r.FormValue("difficulty"),
			Servings:    servings,
			VideoURL:    r.FormValue("videoUrl"),
			Notes:       r.FormValue("notes"),
			CreatedAt:   time.Now().Unix(),
			Views:       0,
		}

		normalizeRecipeSlices(&recipe)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := app.DB.InsertOne(
			ctx,
			recipeCollection,
			recipe,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipe)
	}
}

// --- Update Recipe ---

func UpdateRecipe(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		updates := bson.M{
			"title":       r.FormValue("title"),
			"description": r.FormValue("description"),
			"cookTime":    r.FormValue("cookTime"),
			"cuisine":     r.FormValue("cuisine"),
			"dietary":     splitCSV(r.FormValue("dietary")),
			"portionSize": r.FormValue("portionSize"),
			"season":      r.FormValue("season"),
			"tags":        splitCSV(r.FormValue("tags")),
			"steps":       splitLines(r.FormValue("steps")),
			"difficulty":  r.FormValue("difficulty"),
			"videoUrl":    r.FormValue("videoUrl"),
			"notes":       r.FormValue("notes"),
		}

		if v := r.FormValue("servings"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				updates["servings"] = parsed
			}
		}

		names := r.MultipartForm.Value["ingredientName[]"]
		itemIDs := r.MultipartForm.Value["ingredientItemId[]"]
		types := r.MultipartForm.Value["ingredientType[]"]
		quantities := r.MultipartForm.Value["ingredientQuantity[]"]
		units := r.MultipartForm.Value["ingredientUnit[]"]
		rawAlts := r.MultipartForm.Value["ingredientAlternatives[]"]

		ingredients := make([]models.Ingredient, 0, len(names))

		for i := range names {
			if names[i] == "" || i >= len(quantities) || i >= len(units) {
				continue
			}

			qty, err := strconv.ParseFloat(quantities[i], 64)
			if err != nil {
				continue
			}

			ingredient := models.Ingredient{
				Name:         names[i],
				ItemID:       getSafe(itemIDs, i),
				Type:         getSafe(types, i),
				Quantity:     qty,
				Unit:         units[i],
				Alternatives: []models.IngredientAlternative{},
			}

			if i < len(rawAlts) && rawAlts[i] != "" {
				for _, alt := range strings.Split(rawAlts[i], ",") {
					parts := strings.Split(alt, "|")
					if len(parts) >= 3 {
						ingredient.Alternatives = append(
							ingredient.Alternatives,
							models.IngredientAlternative{
								Name:   parts[0],
								ItemID: parts[1],
								Type:   parts[2],
							},
						)
					}
				}
			}

			ingredients = append(ingredients, ingredient)
		}

		updates["ingredients"] = ingredients

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := app.DB.Update(
			ctx,
			recipeCollection,
			bson.M{"recipeid": id},
			bson.M{"$set": updates},
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"updated"}`))
	}
}

// --- Delete Recipe ---

func DeleteRecipe(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		dels.DeleteRecipe(app)
	}
}
