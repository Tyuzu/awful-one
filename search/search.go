package search

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

// SearchAutocomplete handles autocomplete suggestions based on prefix
// Endpoint: GET /api/v1/ac?prefix={query}
func SearchAutocomplete(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
		if prefix == "" {
			utils.RespondWithJSON(w, http.StatusOK, []string{})
			return
		}

		suggestions, err := GetAutocompleteSuggestions(ctx, app, prefix)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch suggestions")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, suggestions)
	}
}

// SearchByType handles search for a specific entity type
// Endpoint: GET /api/v1/search/{tabId}?query={query}
func SearchByType(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		tabId := strings.TrimSpace(ps.ByName("tabId"))
		query := strings.TrimSpace(r.URL.Query().Get("query"))

		if query == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Query parameter is required")
			return
		}

		// Handle "all" tab - returns object with all entity types
		if tabId == "all" {
			results, err := SearchAll(ctx, app, query)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to search")
				return
			}
			utils.RespondWithJSON(w, http.StatusOK, results)
			return
		}

		// Handle specific entity type search
		results, err := SearchByEntity(ctx, app, tabId, query)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to search %s", tabId))
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, results)
	}
}
