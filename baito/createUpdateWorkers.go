package baito

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/* -------------------- Helpers -------------------- */

// parseWorkerForm parses form data for create or update
func parseWorkerForm(r *http.Request, isUpdate bool) (models.BaitoWorker, bson.M, error) {
	var worker models.BaitoWorker
	update := bson.M{"$set": bson.M{}}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return worker, update, err
	}
	defer r.MultipartForm.RemoveAll()

	ageStr := r.FormValue("age")
	age, _ := strconv.Atoi(ageStr)

	preferredRoles := r.FormValue("roles")
	roles := []string{}
	if preferredRoles != "" {
		for _, r := range strings.Split(preferredRoles, ",") {
			if trimmed := strings.TrimSpace(r); trimmed != "" {
				roles = append(roles, trimmed)
			}
		}
	}

	if isUpdate {
		set := update["$set"].(bson.M)
		set["name"] = r.FormValue("name")
		set["age"] = age
		set["phone"] = r.FormValue("phone")
		set["location"] = r.FormValue("location")
		set["preferredRoles"] = roles
		set["bio"] = r.FormValue("bio")
		set["email"] = r.FormValue("email")
		set["experience"] = r.FormValue("experience")
		set["skills"] = r.FormValue("skills")
		set["availability"] = r.FormValue("availability")
		set["expectedWage"] = r.FormValue("expected_wage")
		set["languages"] = r.FormValue("languages")
		set["updatedAt"] = time.Now().Unix()
	} else {
		worker = models.BaitoWorker{
			UserID:       utils.GetUserIDFromRequest(r),
			BaitoUserID:  utils.GenerateRandomString(12),
			Name:         r.FormValue("name"),
			Age:          age,
			Phone:        r.FormValue("phone"),
			Location:     r.FormValue("location"),
			Preferred:    roles,
			Bio:          r.FormValue("bio"),
			Email:        r.FormValue("email"),
			Experience:   r.FormValue("experience"),
			Skills:       r.FormValue("skills"),
			Availability: r.FormValue("availability"),
			ExpectedWage: r.FormValue("expected_wage"),
			Languages:    r.FormValue("languages"),
			CreatedAt:    time.Now().Unix(),
		}
	}

	return worker, update, nil
}

/* -------------------- Handlers -------------------- */

// CreateWorkerProfile handles creating a new worker profile
func CreateWorkerProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		userID := utils.GetUserIDFromRequest(r)

		// Check if worker profile already exists
		var existing models.BaitoWorker
		err := app.DB.FindOne(
			ctx,
			BaitoWorkersCollection,
			bson.M{"userId": userID},
			&existing,
		)
		if err == nil {
			utils.RespondWithError(w, http.StatusConflict, "Worker profile already exists")
			return
		}
		if err != mongo.ErrNoDocuments {
			log.Printf("DB error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Database error")
			return
		}

		worker, _, err := parseWorkerForm(r, false)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data")
			return
		}

		if worker.Name == "" ||
			worker.Age < 16 ||
			worker.Phone == "" ||
			worker.Location == "" ||
			len(worker.Preferred) == 0 ||
			worker.Bio == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing required fields")
			return
		}

		worker.UserID = userID
		worker.BaitoUserID = userID

		if err = app.DB.Insert(ctx, BaitoWorkersCollection, worker); err != nil {
			log.Printf("Insert error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save worker profile")
			return
		}

		// update user role (non-fatal)
		_ = app.DB.AddToSet(
			ctx,
			UsersCollection,
			bson.M{"userid": userID},
			"role",
			"worker",
		)

		_ = app.DB.UpdateOne(
			ctx,
			UsersCollection,
			bson.M{"userid": userID},
			bson.M{"updated_at": time.Now()},
		)

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Worker profile created successfully",
		})
	}
}

// UpdateWorkerProfile handles updating an existing worker profile
func UpdateWorkerProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		userID := utils.GetUserIDFromRequest(r)
		workerID := ps.ByName("id")

		_, update, err := parseWorkerForm(r, true)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data")
			return
		}

		filter := bson.M{
			"baitoUserId": workerID,
			"userId":      userID,
		}

		err = app.DB.UpdateOne(ctx, BaitoWorkersCollection, filter, update)
		if err != nil {
			log.Printf("Update error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update worker profile")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message":  "Worker profile updated successfully",
			"workerId": workerID,
		})
	}
}
