package mechat

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

//
// ================= HELPERS =================
//

func getUser(r *http.Request) string {
	return utils.GetUserIDFromRequest(r)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	http.Error(w, msg, code)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ensureChatAccess(ctx context.Context, app *infra.Deps, chatID, user string) error {
	return app.DB.FindOne(ctx, MereChatCollection, map[string]any{
		"chatid":       chatID,
		"participants": user,
	}, &struct{}{})
}

func updateLastMessage(
	ctx context.Context,
	app *infra.Deps,
	chatID string,
	msg *models.Message,
) {
	if msg == nil {
		return
	}

	preview := models.MessagePreview{
		Text:      msg.Content,
		UserID:    msg.UserID,
		Timestamp: msg.CreatedAt,
	}

	_ = app.DB.UpdateOne(ctx,
		MereChatCollection,
		map[string]any{"chatid": chatID},
		map[string]any{
			"lastMessage": preview,
			"updatedAt":   time.Now(),
		},
	)
}

//
// ================= MESSAGES =================
//

// Send message (REST)
func SendMessageREST(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)
		chatID := ps.ByName("chatid")

		if err := ensureChatAccess(ctx, app, chatID, user); err != nil {
			writeErr(w, 404, "not found or access denied")
			return
		}

		var body struct {
			Content  string `json:"content"`
			ClientID string `json:"clientId,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, 400, "invalid body")
			return
		}

		body.Content = strings.TrimSpace(body.Content)
		if body.Content == "" {
			writeErr(w, 400, "content required")
			return
		}

		msg := &models.Message{
			MessageID: utils.GenerateRandomDigitString(16),
			ChatID:    chatID,
			UserID:    user,
			Content:   body.Content,
			CreatedAt: time.Now(),
			ReadBy:    []string{user},
			Status:    "sent",
		}

		if err := app.DB.InsertOne(ctx, MessagesCollection, msg); err != nil {
			writeErr(w, 500, "failed to persist message")
			return
		}

		updateLastMessage(ctx, app, chatID, msg)

		resp := struct {
			*models.Message
			ClientID string `json:"clientId,omitempty"`
		}{msg, body.ClientID}

		writeJSON(w, 200, resp)
	}
}

// Edit message
func EditMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)

		id := ps.ByName("messageid")

		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, 400, "invalid body")
			return
		}

		body.Content = strings.TrimSpace(body.Content)
		if body.Content == "" {
			writeErr(w, 400, "content required")
			return
		}

		now := time.Now()
		var msg models.Message

		filter := map[string]any{
			"messageid": id,
			"sender":    user,                        // match actual DB field
			"deleted":   map[string]any{"$ne": true}, // proper Mongo operator
		}

		update := map[string]any{
			"content":  body.Content,
			"editedAt": now,
		}

		if err := app.DB.FindOneAndUpdate(ctx, MessagesCollection, filter, update, &msg); err != nil {
			writeErr(w, 403, "forbidden or not found")
			return
		}

		// Update last message in chat
		updateLastMessage(ctx, app, msg.ChatID, &msg)

		w.WriteHeader(http.StatusNoContent)
	}
}

// Delete message (soft)
func DeleteMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)

		msgID := ps.ByName("messageid")
		if msgID == "" {
			writeErr(w, 400, "invalid message id")
			return
		}

		var msg models.Message
		// Match by string messageid and current user
		if err := app.DB.FindOneAndUpdate(
			ctx,
			MessagesCollection,
			map[string]any{
				"messageid": msgID,
				"userid":    user,
			},
			map[string]any{"deleted": true},
			&msg,
		); err != nil {
			writeErr(w, 403, "forbidden or not found")
			return
		}

		// Update last message if it belongs to this user
		_ = app.DB.UpdateOne(
			ctx,
			MereChatCollection,
			map[string]any{
				"chatid":               msg.ChatID,
				"lastMessage.senderId": msg.UserID,
			},
			map[string]any{"lastMessage": nil},
		)

		w.WriteHeader(http.StatusNoContent)
	}
}

//
// ================= READ STATUS =================
//

func MarkAsRead(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)

		id := ps.ByName("messageid")

		if err := app.DB.AddToSet(
			ctx,
			MessagesCollection,
			map[string]any{"messageid": id},
			"readBy",
			user,
		); err != nil {
			writeErr(w, 404, "not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Unread count per chat
func GetUnreadCount(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)

		var messages []models.Message
		_ = app.DB.FindMany(ctx, MessagesCollection, map[string]any{
			"deleted_ne": true,
			"userid_ne":  user,
			"readBy_ne":  user,
		}, &messages)

		counts := make(map[string]int64)
		for _, m := range messages {
			counts[m.ChatID]++
		}

		var chats []models.Chat
		_ = app.DB.FindMany(ctx, MereChatCollection, map[string]any{
			"participants": user,
		}, &chats)

		type resp struct {
			ChatID string `json:"chatid"`
			Count  int64  `json:"count"`
		}

		out := make([]resp, 0, len(chats))
		for _, c := range chats {
			out = append(out, resp{
				ChatID: c.ChatID,
				Count:  counts[c.ChatID],
			})
		}

		writeJSON(w, 200, out)
	}
}

func SearchMessages(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)
		chatID := ps.ByName("chatid")

		if err := ensureChatAccess(ctx, app, chatID, user); err != nil {
			writeErr(w, 404, "not found or access denied")
			return
		}

		term := strings.TrimSpace(r.URL.Query().Get("term"))

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := parseInt(l); err == nil && v > 0 {
				limit = v
			}
		}

		skip := 0
		if s := r.URL.Query().Get("skip"); s != "" {
			if v, err := parseInt(s); err == nil && v >= 0 {
				skip = v
			}
		}

		filter := map[string]any{
			"chatid":     chatID,
			"deleted_ne": true,
		}
		if term != "" {
			filter["content_contains"] = term
		}

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var msgs []models.Message
		if err := app.DB.FindManyWithOptions(ctx, MessagesCollection, filter, opts, &msgs); err != nil {
			writeErr(w, 500, "internal error")
			return
		}

		if msgs == nil {
			msgs = make([]models.Message, 0)
		}

		writeJSON(w, 200, msgs)
	}
}

// helper
func parseInt(s string) (int, error) {
	v, err := strconv.Atoi(s)
	return v, err
}
