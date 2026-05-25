package newchat

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func GetChat(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		chatID := ps.ByName("chatid")
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var chat models.Chat
		err := app.DB.FindOne(ctx, chatsCollection, map[string]any{
			"chatid": chatID,
			"users":  map[string]any{"$in": []string{userID}},
		}, &chat)
		if err != nil {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}

		var messages []models.Message
		opts := db.FindManyOptions{
			Sort: bson.D{{Key: "createdAt", Value: 1}},
		}
		if err := app.DB.FindManyWithOptions(ctx, messagesCollection, map[string]any{
			"chatid": chatID,
		}, opts, &messages); err != nil {
			http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"chatid":   chatID,
			"messages": messages,
		})
	}
}

func CreateMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		chatID := ps.ByName("chatid")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var chat models.Chat
		if err := app.DB.FindOne(ctx, chatsCollection, map[string]any{"chatid": chatID}, &chat); err != nil {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		isParticipant := false
		for _, uid := range chat.Users {
			if uid == userID {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		text := r.FormValue("text")
		var replyRef *models.ReplyRef
		if v := r.FormValue("replyTo"); v != "" {
			var rr models.ReplyRef
			if err := json.Unmarshal([]byte(v), &rr); err != nil {
				http.Error(w, "Invalid replyTo payload", http.StatusBadRequest)
				return
			}
			replyRef = &rr
		}

		var fileType string
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			files := r.MultipartForm.File["file"]
			if len(files) > 0 {
				ct := files[0].Header.Get("Content-Type")
				switch {
				case strings.HasPrefix(ct, "image/"):
					fileType = "image"
				case strings.HasPrefix(ct, "video/"):
					fileType = "video"
				case strings.HasPrefix(ct, "application/"):
					fileType = "document"
				default:
					http.Error(w, "Unsupported file type", http.StatusBadRequest)
					return
				}
			}
		}

		if text == "" && replyRef == nil && fileType == "" {
			http.Error(w, "No content provided", http.StatusBadRequest)
			return
		}

		now := time.Now()
		msg := models.Message{
			MessageID: utils.GenerateRandomDigitString(16),
			ChatID:    chatID,
			UserID:    userID,
			Text:      text,
			FileType:  fileType,
			CreatedAt: now,
			ReplyTo:   replyRef,
		}

		if err := app.DB.InsertOne(ctx, messagesCollection, msg); err != nil {
			http.Error(w, "Insert failed", http.StatusInternalServerError)
			return
		}

		update := map[string]any{
			"$set": map[string]any{
				"lastMessage": models.MessagePreview{
					Text:      text,
					UserID:    userID,
					Timestamp: now,
				},
				"updatedAt": now,
			},
		}
		_ = app.DB.UpdateOne(ctx, chatsCollection, map[string]any{"chatid": chatID}, update)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msg)
	}
}

func UpdateMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		msgID := ps.ByName("msgid")
		userID := utils.GetUserIDFromRequest(r)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var message models.Message
		if err := app.DB.FindOne(ctx, messagesCollection, map[string]any{"messageid": msgID}, &message); err != nil {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}

		if message.UserID != userID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var input struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Text == "" {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		update := map[string]any{"$set": map[string]any{"text": input.Text}}
		if err := app.DB.UpdateOne(ctx, messagesCollection, map[string]any{"messageid": msgID}, update); err != nil {
			http.Error(w, "Update failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func DeletesMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		msgID := ps.ByName("msgid")
		userID := utils.GetUserIDFromRequest(r)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var message models.Message
		if err := app.DB.FindOne(ctx, messagesCollection, map[string]any{"messageid": msgID}, &message); err != nil {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}

		if message.UserID != userID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if _, err := app.DB.DeleteOne(ctx, messagesCollection, map[string]any{"messageid": msgID}); err != nil {
			http.Error(w, "Delete failed", http.StatusInternalServerError)
			return
		}

		_ = app.DB.UpdateOne(ctx, chatsCollection, map[string]any{"chatid": message.ChatID},
			map[string]any{"$set": map[string]any{"updatedAt": time.Now()}})

		w.WriteHeader(http.StatusOK)
	}
}

func InitChat(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var body struct {
			UserA string `json:"userA"`
			UserB string `json:"userB"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		users := []string{body.UserA, body.UserB}
		sort.Strings(users)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.Chat
		if err := app.DB.FindOne(ctx, chatsCollection, map[string]any{"users": users}, &existing); err == nil {
			json.NewEncoder(w).Encode(map[string]any{"chatid": existing.ChatID})
			return
		}

		chat := models.Chat{
			ChatID: utils.GenerateRandomDigitString(16),
			Users:  users,
		}

		if err := app.DB.InsertOne(ctx, chatsCollection, chat); err != nil {
			http.Error(w, "Failed to create chat", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"chatid": chat.ChatID})
	}
}

func GetUserChats(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var chats []models.Chat
		opts := db.FindManyOptions{
			Sort:  bson.D{{Key: "updatedAt", Value: -1}},
			Limit: 15,
		}
		if err := app.DB.FindManyWithOptions(ctx, chatsCollection, map[string]any{"users": map[string]any{"$in": []string{userID}}}, opts, &chats); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if chats == nil {
			chats = []models.Chat{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chats)
	}
}
