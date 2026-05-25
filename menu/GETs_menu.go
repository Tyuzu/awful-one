package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Fetch a single menu item
func GetMenu(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		placeID := ps.ByName("placeid")
		menuID := ps.ByName("menuid")
		cacheKey := fmt.Sprintf("menu:%s:%s", placeID, menuID)

		// Check cache first
		cachedMenu, err := app.Cache.Get(ctx, cacheKey)
		if err == nil && len(cachedMenu) != 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cachedMenu))
			return
		}

		var menu models.Menu
		err = app.DB.FindOne(ctx, menuCollection, map[string]string{
			"placeid": placeID,
			"menuid":  menuID,
		}, &menu)
		if err != nil {
			http.Error(w, fmt.Sprintf("Menu not found: %v", err), http.StatusNotFound)
			return
		}

		menuJSON, _ := json.Marshal(menu)
		app.Cache.Set(ctx, cacheKey, menuJSON, 1*time.Hour)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(menu)
	}
}

// Fetch stock of a single menu
func GetStock(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		placeID := ps.ByName("placeid")
		menuID := ps.ByName("menuid")

		var menu models.Menu
		err := app.DB.FindOne(r.Context(), menuCollection, map[string]string{
			"placeid": placeID,
			"menuid":  menuID,
		}, &menu)
		if err != nil {
			http.Error(w, fmt.Sprintf("Menu not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(menu)
	}
}

// Fetch all menus for a place
func GetMenus(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var menus []models.Menu
		err := app.DB.FindMany(ctx, menuCollection, map[string]string{
			"placeid": ps.ByName("placeid"),
		}, &menus)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch menus")
			return
		}

		if menus == nil {
			menus = []models.Menu{}
		}
		utils.RespondWithJSON(w, http.StatusOK, menus)
	}
}
