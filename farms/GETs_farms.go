package farms

import (
	"context"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* ---------------------------------------------------- */
/* Get farms selling a specific crop ID                 */
/* ---------------------------------------------------- */

func GetCropFarms(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		cropID := ps.ByName("cropid")
		skip, limit := utils.ParsePagination(r, 10, 100)
		sortBy := r.URL.Query().Get("sortBy")
		sortOrder := r.URL.Query().Get("sortOrder")
		breedFilter := strings.ToLower(r.URL.Query().Get("breed"))

		var crops []models.Crop
		if err := app.DB.FindMany(ctx, cropsCollection, bson.M{"cropid": cropID}, &crops); err != nil || len(crops) == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Crop not found")
			return
		}

		cropName := crops[0].Name
		cropCategory := crops[0].Category

		farmIDs := make([]string, 0, len(crops))
		for _, c := range crops {
			farmIDs = append(farmIDs, c.FarmID)
		}

		var farms []models.Farm
		if err := app.DB.FindMany(
			ctx,
			farmsCollection,
			bson.M{"farmid": bson.M{"$in": farmIDs}},
			&farms,
		); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch farms")
			return
		}

		farmMap := make(map[string]models.Farm, len(farms))
		for _, f := range farms {
			farmMap[f.FarmID] = f
		}

		listings := make([]models.CropListing, 0)
		for _, crop := range crops {
			if breedFilter != "" && strings.ToLower(crop.Notes) != breedFilter {
				continue
			}
			if farm, ok := farmMap[crop.FarmID]; ok {
				listings = append(listings, models.CropListing{
					FarmID:     crop.FarmID,
					FarmName:   farm.Name,
					Location:   farm.Location,
					Breed:      crop.Notes,
					PricePerKg: crop.Price,
					Banner:     crop.Banner,
				})
			}
		}

		sortCropListings(listings, sortBy, sortOrder)

		paginated, page := paginate(listings, skip, limit)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"name":     cropName,
			"category": cropCategory,
			"listings": paginated,
			"total":    len(listings),
			"page":     page,
			"limit":    limit,
		})
	}
}

/* ---------------------------------------------------- */
/* Get farms by crop name (case-insensitive)            */
/* ---------------------------------------------------- */

func GetCropTypeFarms(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		cropName := ps.ByName("cropname")
		if cropName == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing crop name parameter")
			return
		}

		skip, limit := utils.ParsePagination(r, 10, 100)
		sortBy := r.URL.Query().Get("sortBy")
		sortOrder := r.URL.Query().Get("sortOrder")
		breedFilter := strings.ToLower(r.URL.Query().Get("breed"))

		filter := bson.M{
			"name": bson.M{
				"$regex":   "^" + regexp.QuoteMeta(cropName) + "$",
				"$options": "i",
			},
		}

		var crops []models.Crop
		if err := app.DB.FindMany(ctx, cropsCollection, filter, &crops); err != nil || len(crops) == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Crop type not found")
			return
		}

		cropCategory := crops[0].Category

		farmIDs := make([]string, 0, len(crops))
		for _, c := range crops {
			farmIDs = append(farmIDs, c.FarmID)
		}

		var farms []models.Farm
		if err := app.DB.FindMany(
			ctx,
			farmsCollection,
			bson.M{"farmid": bson.M{"$in": farmIDs}},
			&farms,
		); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch farms")
			return
		}

		farmMap := make(map[string]models.Farm, len(farms))
		for _, f := range farms {
			farmMap[f.FarmID] = f
		}

		listings := make([]models.CropListing, 0)
		for _, crop := range crops {
			if breedFilter != "" && strings.ToLower(crop.Notes) != breedFilter {
				continue
			}
			if farm, ok := farmMap[crop.FarmID]; ok {
				var harvestDate string
				if crop.HarvestDate != nil {
					harvestDate = crop.HarvestDate.Format(time.RFC3339)
				}

				listings = append(listings, models.CropListing{
					FarmID:         crop.FarmID,
					CropId:         crop.CropId,
					FarmName:       farm.Name,
					Location:       farm.Location,
					Breed:          crop.Notes,
					PricePerKg:     crop.Price,
					AvailableQtyKg: crop.Quantity,
					HarvestDate:    harvestDate,
					Tags:           farm.Tags,
					Banner:         crop.Banner,
				})
			}
		}

		sortCropListings(listings, sortBy, sortOrder)

		paginated, page := paginate(listings, skip, limit)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"name":     cropName,
			"category": cropCategory,
			"listings": paginated,
			"total":    len(listings),
			"page":     page,
			"limit":    limit,
		})
	}
}

/* ---------------------------------------------------- */
/* Get single farm with crops                           */
/* ---------------------------------------------------- */

func GetFarm(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		id := ps.ByName("id")

		var farm models.Farm
		if err := app.DB.FindOne(ctx, farmsCollection, bson.M{"farmid": id}, &farm); err != nil {
			utils.RespondWithJSON(w, http.StatusNotFound, utils.M{
				"success": false,
				"message": "Farm not found",
			})
			return
		}

		var crops []models.Crop
		_ = app.DB.FindMany(ctx, cropsCollection, bson.M{"farmid": id}, &crops)

		farm.Crops = crops

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"farm":    farm,
		})
	}
}

/* ---------------------------------------------------- */
/* Paginated farms with lookup                          */
/* ---------------------------------------------------- */

func GetPaginatedFarms(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		skip, limit := utils.ParsePagination(r, 10, 100)
		search := r.URL.Query().Get("search")

		pipeline := make([]any, 0)

		if search != "" {
			pipeline = append(pipeline, bson.M{
				"$match": bson.M{
					"$or": []bson.M{
						utils.RegexFilter("name", search),
						utils.RegexFilter("location", search),
						utils.RegexFilter("owner", search),
					},
				},
			})
		}

		pipeline = append(
			pipeline,
			bson.M{"$sort": bson.M{"createdAt": -1}},
			bson.M{"$lookup": bson.M{
				"from":         "crops",
				"localField":   "farmid",
				"foreignField": "farmid",
				"as":           "crops",
			}},
			bson.M{"$skip": skip},
			bson.M{"$limit": limit},
		)

		var farms []models.Farm
		if err := app.DB.Aggregate(ctx, farmsCollection, pipeline, &farms); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error fetching farms")
			return
		}

		total, _ := app.DB.CountDocuments(ctx, farmsCollection, bson.M{})

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"farms":   farms,
			"total":   total,
			"page":    skip/limit + 1,
			"limit":   limit,
		})
	}
}

/* ---------------------------------------------------- */
/* Local helpers                                        */
/* ---------------------------------------------------- */

func sortCropListings(listings []models.CropListing, sortBy, sortOrder string) {
	switch sortBy {
	case "price":
		sort.Slice(listings, func(i, j int) bool {
			if sortOrder == "desc" {
				return listings[i].PricePerKg > listings[j].PricePerKg
			}
			return listings[i].PricePerKg < listings[j].PricePerKg
		})
	case "breed":
		sort.Slice(listings, func(i, j int) bool {
			if sortOrder == "desc" {
				return listings[i].Breed > listings[j].Breed
			}
			return listings[i].Breed < listings[j].Breed
		})
	}
}

func paginate[T any](items []T, skip, limit int) ([]T, int) {
	total := len(items)
	if skip > total {
		skip = total
	}
	end := skip + limit
	if end > total {
		end = total
	}
	page := skip/limit + 1
	return items[skip:end], page
}
