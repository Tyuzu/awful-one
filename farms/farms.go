package farms

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/auditlog"
	"naevis/config/mqevent"
	"naevis/globals"
	"naevis/infra"
	"naevis/middleware"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// --------------------------------------------------
// Create
// --------------------------------------------------

func CreateFarm(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		// SECURITY: Limit multipart form size to prevent DoS (max 50MB)
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			log.Printf("Farm creation: form parse error from %s: %v", r.RemoteAddr, err)
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid form data",
			})
			return
		}

		requestingUserID, ok := ctx.Value(globals.UserIDKey).(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		location := strings.TrimSpace(r.FormValue("location"))
		description := strings.TrimSpace(r.FormValue("description"))
		owner := strings.TrimSpace(r.FormValue("owner"))
		contact := strings.TrimSpace(r.FormValue("contact"))
		availabilityTiming := strings.TrimSpace(r.FormValue("availabilityTiming"))

		// Validation
		if name == "" || location == "" || owner == "" || contact == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Missing required fields",
			})
			return
		}

		// SECURITY: Validate email/phone format
		if !middleware.ValidatePhone(contact) && !middleware.ValidateEmail(contact) {
			log.Printf("Farm creation: invalid contact format from user %s: %s", requestingUserID, contact)
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Invalid contact format. Provide valid email or phone number",
			})
			return
		}

		// Check input length to prevent injection/spam
		if len(name) > 200 || len(location) > 500 || len(description) > 2000 {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Field values too long",
			})
			return
		}

		farm := models.Farm{
			FarmID:             utils.GenerateRandomString(14),
			Name:               name,
			Location:           location,
			Description:        description,
			Owner:              owner,
			Contact:            contact,
			AvailabilityTiming: availabilityTiming,
			Crops:              []models.Crop{},
			CreatedBy:          requestingUserID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := app.DB.InsertOne(ctx, farmsCollection, farm); err != nil {
			log.Printf("Farm creation failed for user %s: %v", requestingUserID, err)
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to create farm",
			})
			return
		}

		// SECURITY: Log audit trail
		go auditlog.LogAction(ctx, app, r, requestingUserID, models.AuditActionFarmCreate,
			"farm", farm.FarmID, "success", map[string]interface{}{
				"name":     farm.Name,
				"location": farm.Location,
			})

		/* -------- Publish FarmCreated Event -------- */
		farmPayload := mqevent.FarmCreatedPayload{
			FarmID:     farm.FarmID,
			UserID:     requestingUserID,
			FarmName:   farm.Name,
			Location:   farm.Location,
			OccurredAt: time.Now(),
		}

		farmBytes, err := json.Marshal(farmPayload)
		if err == nil {
			publishCtx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			_ = app.MQ.Publish(
				publishCtx,
				mqevent.FarmCreated,
				farmBytes,
			)
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"id":      farm.FarmID,
		})
	}
}

// --------------------------------------------------
// Edit
// --------------------------------------------------

func EditFarm(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		farmID := ps.ByName("id")

		if farmID == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Missing farm id",
			})
			return
		}

		if _, ok := ctx.Value(globals.UserIDKey).(string); !ok {
			http.Error(w, "Invalid user", http.StatusBadRequest)
			return
		}

		update := bson.M{}
		contentType := r.Header.Get("Content-Type")

		var input models.Farm

		if strings.HasPrefix(contentType, "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
					"success": false,
					"message": "Malformed multipart data",
				})
				return
			}

			input.Name = r.FormValue("name")
			input.Location = r.FormValue("location")
			input.Description = r.FormValue("description")
			input.Owner = r.FormValue("owner")
			input.Contact = r.FormValue("contact")
			input.AvailabilityTiming = r.FormValue("availabilityTiming")
		} else {
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
					"success": false,
					"message": "Invalid JSON body",
				})
				return
			}
		}

		if input.Name != "" {
			update["name"] = input.Name
		}
		if input.Location != "" {
			update["location"] = input.Location
		}
		if input.Description != "" {
			update["description"] = input.Description
		}
		if input.Owner != "" {
			update["owner"] = input.Owner
		}
		if input.Contact != "" {
			update["contact"] = input.Contact
		}
		if input.AvailabilityTiming != "" {
			update["availabilityTiming"] = input.AvailabilityTiming
		}

		if len(update) == 0 {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "No fields to update",
			})
			return
		}

		update["updatedAt"] = time.Now()

		userID, _ := ctx.Value(globals.UserIDKey).(string)

		if err := app.DB.UpdateOne(
			ctx,
			farmsCollection,
			bson.M{"farmid": farmID},
			bson.M{"$set": update},
		); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Database error",
			})
			return
		}

		/* -------- Publish FarmUpdated Event -------- */
		farmUpdatePayload := mqevent.FarmUpdatedPayload{
			FarmID:     farmID,
			UserID:     userID,
			OccurredAt: time.Now(),
		}

		farmUpdateBytes, err := json.Marshal(farmUpdatePayload)
		if err == nil {
			publishCtx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			_ = app.MQ.Publish(
				publishCtx,
				mqevent.FarmUpdated,
				farmUpdateBytes,
			)
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"message": "Farm updated",
		})
	}
}

// --------------------------------------------------
// Delete
// --------------------------------------------------

func DeleteFarm(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		farmID := ps.ByName("id")

		if farmID == "" {
			utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
				"success": false,
				"message": "Missing farm id",
			})
			return
		}

		if _, ok := ctx.Value(globals.UserIDKey).(string); !ok {
			http.Error(w, "Invalid user", http.StatusBadRequest)
			return
		}

		if _, err := app.DB.DeleteOne(ctx, farmsCollection, bson.M{"farmid": farmID}); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
			})
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
		})
	}
}
