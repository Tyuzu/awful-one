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

// CreateNotification creates a new notification for a user
func CreateNotification(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var body struct {
			UserID      string `json:"userId"`
			Type        string `json:"type"`
			Title       string `json:"title"`
			Message     string `json:"message"`
			EntityType  string `json:"entityType"`
			EntityID    string `json:"entityId"`
			RelatedUser string `json:"relatedUser"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		// Validate required fields
		body.UserID = strings.TrimSpace(body.UserID)
		body.Type = strings.TrimSpace(body.Type)
		body.Message = strings.TrimSpace(body.Message)

		if body.UserID == "" || body.Type == "" || body.Message == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing required fields: userId, type, message")
			return
		}

		notification := models.Notification{
			ID:          primitive.NewObjectID().Hex(),
			UserID:      body.UserID,
			Type:        body.Type,
			Title:       strings.TrimSpace(body.Title),
			Message:     body.Message,
			EntityType:  strings.TrimSpace(body.EntityType),
			EntityID:    strings.TrimSpace(body.EntityID),
			RelatedUser: strings.TrimSpace(body.RelatedUser),
			IsRead:      false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := app.DB.Insert(ctx, notificationsCollection, notification); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create notification")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, notification)
	}
}

// BulkCreateNotifications creates multiple notifications at once
func BulkCreateNotifications(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var body struct {
			Notifications []struct {
				UserID      string `json:"userId"`
				Type        string `json:"type"`
				Title       string `json:"title"`
				Message     string `json:"message"`
				EntityType  string `json:"entityType"`
				EntityID    string `json:"entityId"`
				RelatedUser string `json:"relatedUser"`
			} `json:"notifications"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if len(body.Notifications) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "No notifications provided")
			return
		}

		notifications := make([]interface{}, len(body.Notifications))
		for i, n := range body.Notifications {
			notifications[i] = models.Notification{
				ID:          primitive.NewObjectID().Hex(),
				UserID:      strings.TrimSpace(n.UserID),
				Type:        strings.TrimSpace(n.Type),
				Title:       strings.TrimSpace(n.Title),
				Message:     strings.TrimSpace(n.Message),
				EntityType:  strings.TrimSpace(n.EntityType),
				EntityID:    strings.TrimSpace(n.EntityID),
				RelatedUser: strings.TrimSpace(n.RelatedUser),
				IsRead:      false,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
		}

		if err := app.DB.InsertMany(ctx, notificationsCollection, notifications); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create notifications")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"inserted": len(notifications),
		})
	}
}

// GetUserNotifications retrieves all notifications for a user
func GetUserNotifications(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := ps.ByName("userid")
		userID = strings.TrimSpace(userID)
		if userID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		// Get unread only query param
		unreadOnly := r.URL.Query().Get("unread") == "true"

		filter := bson.M{"userId": userID}
		if unreadOnly {
			filter["isRead"] = false
		}

		var notifications []models.Notification
		if err := app.DB.FindMany(ctx, notificationsCollection, filter, &notifications); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch notifications")
			return
		}

		// If notifications is nil, return empty array instead of null
		if notifications == nil {
			notifications = []models.Notification{}
		}

		utils.RespondWithJSON(w, http.StatusOK, notifications)
	}
}

// GetUnreadCount gets count of unread notifications
func GetUnreadCount(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := ps.ByName("userid")
		userID = strings.TrimSpace(userID)
		if userID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		filter := bson.M{"userId": userID, "isRead": false}
		count, err := app.DB.CountDocuments(ctx, notificationsCollection, filter)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to count notifications")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"count": count,
		})
	}
}

// MarkAsRead marks a notification as read
func MarkAsRead(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		notificationID := strings.TrimSpace(ps.ByName("notificationid"))
		if notificationID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid notification ID")
			return
		}

		filter := bson.M{"_id": notificationID}
		update := bson.M{
			"$set": bson.M{
				"isRead":    true,
				"updatedAt": time.Now(),
			},
		}

		if err := app.DB.UpdateOne(ctx, notificationsCollection, filter, update); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update notification")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"updated": true,
		})
	}
}

// MarkAllAsRead marks all notifications as read for a user
func MarkAllAsRead(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := ps.ByName("userid")
		userID = strings.TrimSpace(userID)
		if userID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		filter := bson.M{"userId": userID, "isRead": false}
		update := bson.M{
			"$set": bson.M{
				"isRead":    true,
				"updatedAt": time.Now(),
			},
		}

		if err := app.DB.UpdateMany(ctx, notificationsCollection, filter, update); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update notifications")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"updated": true,
		})
	}
}

// DeleteNotification deletes a notification
func DeleteNotification(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		notificationID := strings.TrimSpace(ps.ByName("notificationid"))
		if notificationID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid notification ID")
			return
		}

		filter := bson.M{"_id": notificationID}
		if _, err := app.DB.DeleteOne(ctx, notificationsCollection, filter); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete notification")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deleted": true,
		})
	}
}

// ClearAllNotifications deletes all notifications for a user
func ClearAllNotifications(app *infra.Deps) httprouter.Handle {
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
		if err := app.DB.DeleteMany(ctx, notificationsCollection, filter); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete notifications")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deleted": true,
		})
	}
}
