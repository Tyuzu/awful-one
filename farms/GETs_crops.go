package farms

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* ---------------------------------------------------- */
/* Filtered Crops                                       */
/* ---------------------------------------------------- */

func GetFilteredCrops(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		params := r.URL.Query()
		filter := bson.M{}

		if c := params.Get("category"); c != "" {
			filter["category"] = c
		}
		if region := params.Get("region"); region != "" {
			filter["farmLocation"] = region
		}
		if params.Get("inStock") == "true" {
			filter["quantity"] = bson.M{"$gt": 0}
		}

		price := bson.M{}
		if min := utils.ParseFloat(params.Get("minPrice")); min > 0 {
			price["$gte"] = min
		}
		if max := utils.ParseFloat(params.Get("maxPrice")); max > 0 {
			price["$lte"] = max
		}
		if len(price) > 0 {
			filter["price"] = price
		}

		var crops []models.Crop
		if err := app.DB.FindMany(ctx, cropsCollection, filter, &crops); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch crops")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"crops":   crops,
		})
	}
}

/* ---------------------------------------------------- */
/* Pre Crop Catalogue (Redis → DB → CSV)                */
/* ---------------------------------------------------- */

func GetPreCropCatalogue(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		const cacheKey = "crop_catalogue"
		var crops []models.CropCatalogueItem

		/* ------------------------------------------------ */
		/* 1. Cache                                         */
		/* ------------------------------------------------ */

		if data, err := app.Cache.Get(ctx, cacheKey); err == nil && len(data) > 0 {
			if json.Unmarshal(data, &crops) == nil {
				utils.RespondWithJSON(w, http.StatusOK, utils.M{
					"success": true,
					"crops":   crops,
				})
				return
			}
		}

		/* ------------------------------------------------ */
		/* 2. Database                                      */
		/* ------------------------------------------------ */

		if err := app.DB.FindMany(ctx, catalogueCollection, bson.M{}, &crops); err == nil && len(crops) > 0 {
			if jsonBytes, err := json.Marshal(crops); err == nil {
				_ = app.Cache.Set(ctx, cacheKey, jsonBytes, 2*time.Hour)
			}

			utils.RespondWithJSON(w, http.StatusOK, utils.M{
				"success": true,
				"crops":   crops,
			})
			return
		}

		/* ------------------------------------------------ */
		/* 3. CSV fallback                                  */
		/* ------------------------------------------------ */

		file, err := os.Open("data/pre_crop_catalogue.csv")
		if err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to retrieve catalogue",
			})
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		headers, err := reader.Read()
		if err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Invalid CSV",
			})
			return
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil || len(record) != len(headers) {
				continue
			}

			item := models.CropCatalogueItem{}
			for i, h := range headers {
				switch strings.ToLower(h) {
				case "name":
					item.Name = record[i]
				case "category":
					item.Category = record[i]
				case "banner":
					item.Banner = record[i]
				case "stock":
					item.Stock, _ = strconv.Atoi(record[i])
				case "unit":
					item.Unit = record[i]
				case "featured":
					item.Featured = strings.ToLower(record[i]) == "true"
				case "pricerange":
					parts := strings.Split(record[i], "-")
					if len(parts) == 2 {
						min, _ := strconv.Atoi(parts[0])
						max, _ := strconv.Atoi(parts[1])
						item.PriceRange = []int{min, max}
					}
				}
			}

			crops = append(crops, item)
		}

		if jsonBytes, err := json.Marshal(crops); err == nil {
			_ = app.Cache.Set(ctx, cacheKey, jsonBytes, 2*time.Hour)
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"crops":   crops,
		})
	}
}

/* ---------------------------------------------------- */
/* Crop Catalogue (unique by name + catalogueId)        */
/* ---------------------------------------------------- */

func GetCropCatalogue(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var allCrops []models.Crop
		if err := app.DB.FindMany(ctx, cropsCollection, bson.M{}, &allCrops); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to fetch crop catalogue",
			})
			return
		}

		seen := make(map[string]bool)
		unique := make([]models.Crop, 0, len(allCrops))

		for _, c := range allCrops {
			key := strings.ToLower(c.Name + c.CatalogueId)
			if !seen[key] {
				seen[key] = true
				unique = append(unique, c)
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"crops":   unique,
		})
	}
}

/* ---------------------------------------------------- */
/* Crop Types (grouped in application layer)             */
/* ---------------------------------------------------- */

func GetCropTypes(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var crops []models.Crop
		if err := app.DB.FindMany(ctx, cropsCollection, bson.M{}, &crops); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to fetch crops",
			})
			return
		}

		type agg struct {
			Name           string
			MinPrice       float64
			MaxPrice       float64
			AvailableCount int
			Unit           string
			Banner         string
		}

		m := map[string]*agg{}

		for _, c := range crops {
			a, ok := m[c.Name]
			if !ok {
				m[c.Name] = &agg{
					Name:     c.Name,
					MinPrice: c.Price,
					MaxPrice: c.Price,
					Unit:     c.Unit,
					Banner:   getCropBanner(strings.ToLower(c.Name)),
				}
				a = m[c.Name]
			}

			if c.Price < a.MinPrice {
				a.MinPrice = c.Price
			}
			if c.Price > a.MaxPrice {
				a.MaxPrice = c.Price
			}
			if c.Quantity > 0 {
				a.AvailableCount++
			}
		}

		result := make([]agg, 0, len(m))
		for _, v := range m {
			result = append(result, *v)
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success":   true,
			"cropTypes": result,
		})
	}
}
