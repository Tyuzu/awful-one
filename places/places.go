package places

import (
	"context"
	"fmt"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func parseAndBuildPlace(r *http.Request, mode string) (models.Place, bson.M, error) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		return models.Place{}, nil, fmt.Errorf("unable to parse form")
	}

	place := models.Place{}
	update := bson.M{}

	apply := func(key string, val interface{}) {
		update[key] = val
		switch key {
		case "name":
			place.Name = val.(string)
		case "address":
			place.Address = val.(string)
		case "description":
			place.Description = val.(string)
		case "category":
			place.Category = val.(string)
		case "capacity":
			place.Capacity = val.(int)
		case "city":
			place.City = val.(string)
		case "country":
			place.Country = val.(string)
		case "zipCode":
			place.ZipCode = val.(string)
		case "phone":
			place.Phone = val.(string)
		case "banner":
			place.Banner = val.(string)
		case "placeid":
			place.PlaceID = val.(string)
		case "created_at":
			place.CreatedAt = val.(time.Time)
		case "updated_at":
			place.UpdatedAt = val.(time.Time)
		case "status":
			place.Status = val.(string)
		}
	}

	// required fields
	name := strings.TrimSpace(r.FormValue("name"))
	if mode == "create" && name == "" {
		return models.Place{}, nil, fmt.Errorf("name is required")
	}
	if name != "" {
		apply("name", name)
	}

	address := strings.TrimSpace(r.FormValue("address"))
	if mode == "create" && address == "" {
		return models.Place{}, nil, fmt.Errorf("address is required")
	}
	if address != "" {
		apply("address", address)
	}

	description := strings.TrimSpace(r.FormValue("description"))
	if mode == "create" && description == "" {
		return models.Place{}, nil, fmt.Errorf("description is required")
	}
	if description != "" {
		apply("description", description)
	}

	category := strings.TrimSpace(r.FormValue("category"))
	if mode == "create" && category == "" {
		return models.Place{}, nil, fmt.Errorf("category is required")
	}
	if category != "" {
		apply("category", category)
	}

	capacityStr := strings.TrimSpace(r.FormValue("capacity"))
	if mode == "create" && capacityStr == "" {
		return models.Place{}, nil, fmt.Errorf("capacity is required")
	}
	if capacityStr != "" {
		capacity, err := strconv.Atoi(capacityStr)
		if err != nil || capacity <= 0 {
			return models.Place{}, nil, fmt.Errorf("capacity must be a positive integer")
		}
		apply("capacity", capacity)
	}

	// optional
	if city := strings.TrimSpace(r.FormValue("city")); city != "" {
		apply("city", city)
	}
	if country := strings.TrimSpace(r.FormValue("country")); country != "" {
		apply("country", country)
	}
	if zipcode := strings.TrimSpace(r.FormValue("zipCode")); zipcode != "" {
		apply("zipCode", zipcode)
	}
	if phone := strings.TrimSpace(r.FormValue("phone")); phone != "" {
		apply("phone", phone)
	}

	// timestamps + ids
	if mode == "create" {
		id := utils.GenerateRandomString(14)
		now := time.Now()

		apply("placeid", id)
		apply("created_at", now)
		apply("status", "active")
	} else {
		apply("updated_at", time.Now())
	}

	return place, update, nil
}

// --- CreatePlace endpoint ---
func CreatePlace(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		place, _, err := parseAndBuildPlace(r, "create")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		requestingUserID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Invalid user", http.StatusBadRequest)
			return
		}
		place.CreatedBy = requestingUserID

		// Insert using db.Database interface
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := app.DB.Insert(ctx, placesCollection, place); err != nil {
			http.Error(w, "Error creating place", http.StatusInternalServerError)
			return
		}

		// Set user data
		userdata.SetUserData("place", place.PlaceID, requestingUserID, "", "", app)

		utils.RespondWithJSON(w, http.StatusCreated, place)
	}
}
