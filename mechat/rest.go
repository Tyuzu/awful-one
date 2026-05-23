package mechat

import (
	"encoding/json"
	"net/http"
	"sort"
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
// ================= CHATS =================
//

func StartNewChat(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		user := getUser(r)

		var body struct {
			Participants []string `json:"participants"`
			EntityType   string   `json:"entityType"`
			EntityId     string   `json:"entityId"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			writeErr(w, 400, "invalid body")
			return
		}

		set := map[string]struct{}{user: {}}
		for _, p := range body.Participants {
			if p != "" {
				set[p] = struct{}{}
			}
		}

		participants := make([]string, 0, len(set))
		for p := range set {
			participants = append(participants, p)
		}
		sort.Strings(participants)

		filter := map[string]any{
			"participants": participants,
			"entitytype":   body.EntityType,
			"entityid":     body.EntityId,
		}

		var existing models.Chat
		if err := app.DB.FindOne(ctx, MereChatCollection, filter, &existing); err == nil {
			writeJSON(w, 200, existing)
			return
		}

		now := time.Now()
		chat := models.Chat{
			ChatID:       utils.GenerateRandomString(16),
			Participants: participants,
			EntityType:   body.EntityType,
			EntityId:     body.EntityId,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := app.DB.InsertOne(ctx, MereChatCollection, chat); err != nil {
			writeErr(w, 500, "failed to create chat")
			return
		}

		writeJSON(w, 200, chat)
	}
}

func GetChatMessages(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := utils.GetUserIDFromRequest(r)

		chatID := strings.TrimSpace(ps.ByName("chatid"))
		if chatID == "" {
			writeErr(w, 400, "missing chat id")
			return
		}

		// access check
		if err := app.DB.FindOne(ctx, MereChatCollection, map[string]any{
			"chatid":       chatID,
			"participants": user,
		}, &struct{}{}); err != nil {
			writeErr(w, 404, "not found or access denied")
			return
		}

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				limit = v
			}
		}

		skip := 0
		if s := r.URL.Query().Get("skip"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v >= 0 {
				skip = v
			}
		}

		filter := map[string]any{
			"chatid":     chatID,
			"deleted_ne": true,
		}

		opts := db.FindManyOptions{
			Limit: limit,
			Skip:  skip,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var msgs []models.Message
		if err := app.DB.FindManyWithOptions(ctx, MessagesCollection, filter, opts, &msgs); err != nil {
			writeErr(w, 500, "failed to load messages")
			return
		}

		if msgs == nil {
			msgs = make([]models.Message, 0)
		}

		writeJSON(w, 200, msgs)
	}
}

func GetChatByID(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := utils.GetUserIDFromRequest(r)

		chatID := ps.ByName("chatid")
		var chat models.Chat

		if err := app.DB.FindOne(ctx, MereChatCollection, map[string]any{
			"chatid":       chatID,
			"participants": user,
		}, &chat); err != nil {
			writeErr(w, 404, "not found or access denied")
			return
		}

		writeJSON(w, 200, chat)
	}
}

func GetUserChats(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		user := utils.GetUserIDFromRequest(r)

		skip := 0
		limit := 20

		if v := r.URL.Query().Get("skip"); v != "" {
			if i, err := strconv.Atoi(v); err == nil && i >= 0 {
				skip = i
			}
		}
		if v := r.URL.Query().Get("limit"); v != "" {
			if i, err := strconv.Atoi(v); err == nil && i > 0 {
				limit = i
			}
		}

		opts := db.FindManyOptions{
			Skip:  skip,
			Limit: limit,
			Sort:  bson.D{{Key: "updatedAt", Value: -1}},
		}

		var chats []models.Chat
		if err := app.DB.FindManyWithOptions(
			ctx,
			MereChatCollection,
			map[string]any{"participants": user},
			opts,
			&chats,
		); err != nil {
			writeErr(w, 500, "failed to load chats")
			return
		}

		if chats == nil {
			chats = make([]models.Chat, 0)
		}

		writeJSON(w, 200, chats)
	}
}

func UploadAttachment(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		user := utils.GetUserIDFromRequest(r)
		chatID := ps.ByName("chatid")

		mediaID := r.FormValue("mediaid")
		savedName := r.FormValue("savedname")
		extn := r.FormValue("extn")
		mimeType := r.FormValue("mimeType")
		fileSizeS := r.FormValue("fileSize")

		if mediaID == "" || savedName == "" || mimeType == "" {
			writeErr(w, 400, "invalid upload payload")
			return
		}

		fileSize, _ := strconv.ParseInt(fileSizeS, 10, 64)

		mediaType := "file"
		if strings.HasPrefix(mimeType, "image/") {
			mediaType = "image"
		} else if strings.HasPrefix(mimeType, "video/") {
			mediaType = "video"
		} else if strings.HasPrefix(mimeType, "audio/") {
			mediaType = "audio"
		}

		// authorization check
		if err := app.DB.FindOne(ctx, MereChatCollection, map[string]any{
			"chatid":       chatID,
			"participants": user,
		}, &struct{}{}); err != nil {
			writeErr(w, 403, "chat not found or access denied")
			return
		}

		now := time.Now()

		// FIX: generate MessageID BEFORE insert
		msg := &models.Message{
			MessageID: utils.GenerateRandomDigitString(16),
			ChatID:    chatID,
			UserID:    user,
			Content:   "",
			Media: &models.Media{
				MediaID:   mediaID,
				Type:      mediaType,
				URL:       savedName,
				MimeType:  mimeType,
				FileSize:  fileSize,
				Extn:      extn,
				CreatedAt: now,
			},
			FileURL:   savedName,
			FileType:  mediaType,
			ReadBy:    []string{user},
			Status:    "sent",
			CreatedAt: now,
		}

		if err := app.DB.InsertOne(ctx, MessagesCollection, msg); err != nil {
			writeErr(w, 500, "failed to persist message")
			return
		}

		// update chat metadata (non-critical)
		_ = app.DB.UpdateOne(
			ctx,
			MereChatCollection,
			map[string]any{"chatid": chatID},
			map[string]any{
				"updatedAt": now,
				"lastMessage": map[string]any{
					"text":      "[attachment]",
					"senderId":  user,
					"timestamp": now,
				},
			},
		)

		writeJSON(w, 200, msg)
	}
}
