package moderator

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

type ModeratorApplication struct {
	ID        string    `bson:"id" json:"id"`
	UserID    string    `bson:"userId" json:"userId"`
	Reason    string    `bson:"reason" json:"reason"`
	Status    string    `bson:"status" json:"status"` // pending, approved, rejected
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

func ApplyModerator(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		var payload struct {
			UserID string `json:"userId"`
			Reason string `json:"reason"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, `{"error":"Invalid JSON payload"}`, http.StatusBadRequest)
			return
		}

		payload.UserID = strings.TrimSpace(payload.UserID)
		payload.Reason = strings.TrimSpace(payload.Reason)

		if payload.UserID == "" || payload.Reason == "" {
			http.Error(w, `{"error":"Missing required fields"}`, http.StatusBadRequest)
			return
		}

		// Prevent duplicate applications
		var existing ModeratorApplication
		err := app.DB.FindOne(
			ctx,
			moderatorAppsCollection,
			bson.M{"userId": payload.UserID},
			&existing,
		)
		if err == nil {
			http.Error(w, `{"error":"You have already applied to be a moderator"}`, http.StatusConflict)
			return
		}

		appx := ModeratorApplication{
			ID:        "mod_" + utils.GenerateRandomString(16),
			UserID:    payload.UserID,
			Reason:    payload.Reason,
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
		}

		if err := app.DB.Insert(ctx, moderatorAppsCollection, appx); err != nil {
			http.Error(w, `{"error":"Failed to save application"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Moderator application submitted",
			"id":      appx.ID,
		})
	}
}
