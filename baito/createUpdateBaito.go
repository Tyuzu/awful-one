package baito

import (
	"log"
	"net/http"
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

func parseBaitoForm(r *http.Request, isUpdate bool) (models.Baito, bson.M, error) {
	b := models.Baito{}
	update := bson.M{"$set": bson.M{}}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return b, update, err
	}
	defer r.MultipartForm.RemoveAll()

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	category := strings.TrimSpace(r.FormValue("category"))
	subcategory := strings.TrimSpace(r.FormValue("subcategory"))
	location := strings.TrimSpace(r.FormValue("location"))
	wage := strings.TrimSpace(r.FormValue("wage"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	requirements := strings.TrimSpace(r.FormValue("requirements"))
	workHours := strings.TrimSpace(r.FormValue("workHours"))
	benefits := strings.TrimSpace(r.FormValue("benefits"))
	email := strings.TrimSpace(r.FormValue("email"))
	tagsStr := strings.TrimSpace(r.FormValue("tags"))

	tags := []string{}
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	if isUpdate {
		set := update["$set"].(bson.M)
		set["title"] = title
		set["description"] = description
		set["category"] = category
		set["subcategory"] = subcategory
		set["location"] = location
		set["wage"] = wage
		set["phone"] = phone
		set["requirements"] = requirements
		set["work_hours"] = workHours
		set["benefits"] = benefits
		set["email"] = email
		set["tags"] = tags
		set["updated_at"] = time.Now()
	} else {
		b = models.Baito{
			BaitoId:      utils.GenerateRandomString(15),
			Title:        title,
			Description:  description,
			Category:     category,
			SubCategory:  subcategory,
			Location:     location,
			Wage:         wage,
			Phone:        phone,
			Requirements: requirements,
			WorkHours:    workHours,
			Benefits:     benefits,
			Email:        email,
			Tags:         tags,
			OwnerID:      utils.GetUserIDFromRequest(r),
			CreatedAt:    time.Now(),
		}
	}

	return b, update, nil
}

/* -------------------- Handlers -------------------- */

func CreateBaito(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		b, _, err := parseBaitoForm(r, false)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data")
			return
		}

		if b.Title == "" ||
			b.Description == "" ||
			b.Category == "" ||
			b.SubCategory == "" ||
			b.Location == "" ||
			b.Wage == "" ||
			b.Phone == "" ||
			b.Requirements == "" ||
			b.WorkHours == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing required fields")
			return
		}

		if err := app.DB.Insert(ctx, BaitoCollection, b); err != nil {
			log.Printf("Insert error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save baito")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"baitoid": b.BaitoId,
		})
	}
}

func UpdateBaito(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		_, update, err := parseBaitoForm(r, true)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data")
			return
		}

		filter := bson.M{
			"baitoid": ps.ByName("baitoid"),
			"ownerid": utils.GetUserIDFromRequest(r),
		}

		err = app.DB.UpdateOne(ctx, BaitoCollection, filter, update)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				utils.RespondWithError(w, http.StatusNotFound, "Baito not found or unauthorized")
				return
			}
			log.Printf("Update error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update baito")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "Baito updated",
			"baitoid": ps.ByName("baitoid"),
		})
	}
}
