package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetPreferences gets notification preferences for a user
func GetPreferences(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := ps.ByName("userid")
		userID = strings.TrimSpace(userID)
		if userID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		filter := bson.M{"userId": userID}
		var preference models.NotificationPreference

		err := app.DB.FindOne(ctx, notificationsPreferencesCollection, filter, &preference)
		if err != nil {
			// Return default preferences if not found
			preference = models.NotificationPreference{
				ID:              primitive.NewObjectID().Hex(),
				UserID:          userID,
				MentionsEnabled: true,
				FollowsEnabled:  true,
				CommentsEnabled: true,
				LikesEnabled:    true,
				MessagesEnabled: true,
				AllEnabled:      true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			// Create default preferences
			app.DB.Insert(ctx, notificationsPreferencesCollection, preference)
		}

		utils.RespondWithJSON(w, http.StatusOK, preference)
	}
}

// UpdatePreferences updates notification preferences for a user
func UpdatePreferences(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := ps.ByName("userid")
		userID = strings.TrimSpace(userID)
		if userID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var body struct {
			MentionsEnabled *bool `json:"mentionsEnabled"`
			FollowsEnabled  *bool `json:"followsEnabled"`
			CommentsEnabled *bool `json:"commentsEnabled"`
			LikesEnabled    *bool `json:"likesEnabled"`
			MessagesEnabled *bool `json:"messagesEnabled"`
			AllEnabled      *bool `json:"allEnabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		// If allEnabled is set, update all other settings
		updates := bson.M{
			"updatedAt": time.Now(),
		}

		if body.AllEnabled != nil {
			updates["allEnabled"] = *body.AllEnabled
			updates["mentionsEnabled"] = *body.AllEnabled
			updates["followsEnabled"] = *body.AllEnabled
			updates["commentsEnabled"] = *body.AllEnabled
			updates["likesEnabled"] = *body.AllEnabled
			updates["messagesEnabled"] = *body.AllEnabled
		} else {
			if body.MentionsEnabled != nil {
				updates["mentionsEnabled"] = *body.MentionsEnabled
			}
			if body.FollowsEnabled != nil {
				updates["followsEnabled"] = *body.FollowsEnabled
			}
			if body.CommentsEnabled != nil {
				updates["commentsEnabled"] = *body.CommentsEnabled
			}
			if body.LikesEnabled != nil {
				updates["likesEnabled"] = *body.LikesEnabled
			}
			if body.MessagesEnabled != nil {
				updates["messagesEnabled"] = *body.MessagesEnabled
			}
		}

		filter := bson.M{"userId": userID}
		update := bson.M{
			"$set": updates,
		}

		if err := app.DB.UpdateOne(ctx, notificationsPreferencesCollection, filter, update); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update preferences")
			return
		}

		// Fetch and return updated preferences
		var updated models.NotificationPreference
		app.DB.FindOne(ctx, notificationsPreferencesCollection, filter, &updated)
		utils.RespondWithJSON(w, http.StatusOK, updated)
	}
}
