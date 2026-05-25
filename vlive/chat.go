package vlive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ChatMessage struct {
	LiveID  string    `json:"liveId" bson:"live_id"`
	UserID  string    `json:"userId" bson:"user_id"`
	Message string    `json:"message" bson:"message"`
	SentAt  time.Time `json:"sentAt" bson:"sent_at"`
}

// ----------------------- CHAT -----------------------

func ChatWebSocket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		liveID := ps.ByName("liveid")
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		objID, err := primitive.ObjectIDFromHex(liveID)
		if err != nil {
			writeError(w, "invalid id", http.StatusBadRequest)
			return
		}

		stream, err := getStreamByID(r.Context(), app, objID)
		if err != nil {
			writeError(w, "not found", http.StatusNotFound)
			return
		}

		if !stream.IsPublic && stream.CreatorID != userID {
			allowed := CheckEntityAccess(
				r.Context(),
				app,
				userID,
				stream.EntityType,
				stream.EntityID,
			)
			if !allowed {
				writeError(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if !stream.ChatEnabled {
			writeError(w, "chat disabled for this stream", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		incViewerCount(ctx, app, liveID)
		defer decViewerCount(context.Background(), app, liveID)

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if len(raw) == 0 || len(raw) > 1024 {
				continue
			}

			// reload stream (owner may toggle flags live)
			stream, err = getStreamByID(ctx, app, objID)
			if err != nil {
				_ = conn.WriteMessage(
					websocket.TextMessage,
					[]byte(`{"error":"stream_not_found"}`),
				)
				continue
			}
			if !stream.ChatEnabled {
				_ = conn.WriteMessage(
					websocket.TextMessage,
					[]byte(`{"error":"chat_disabled"}`),
				)
				continue
			}

			if stream.SlowModeSeconds > 0 {
				slowKey := "vlive:slow:" + liveID + ":" + userID
				exists, _ := app.Cache.Exists(ctx, slowKey)
				if exists {
					_ = conn.WriteMessage(
						websocket.TextMessage,
						[]byte(fmt.Sprintf(
							`{"error":"slow_mode_active","retryAfter":%d}`,
							stream.SlowModeSeconds,
						)),
					)
					continue
				}

				_ = app.Cache.Set(
					ctx,
					slowKey,
					[]byte("1"),
					time.Duration(stream.SlowModeSeconds)*time.Second,
				)
			}

			cm := ChatMessage{
				LiveID:  liveID,
				UserID:  userID,
				Message: string(raw),
				SentAt:  time.Now(),
			}

			data, _ := json.Marshal(cm)

			// echo back to sender (no pubsub available)
			_ = conn.WriteMessage(websocket.TextMessage, data)

			go func(cm ChatMessage) {
				_ = app.DB.Insert(
					context.Background(),
					vliveChatsCollection,
					cm,
				)
			}(cm)
		}
	}
}

// ----------------------- CHAT SETTINGS -----------------------

func ChatEnable(app *infra.Deps, enable bool) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		idHex := ps.ByName("liveid")
		id, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			writeError(w, "invalid liveid", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := getStreamByID(r.Context(), app, id)
		if err != nil {
			writeError(w, "stream not found", http.StatusNotFound)
			return
		}
		if !isOwner(userID, stream) {
			writeError(w, "forbidden", http.StatusForbidden)
			return
		}

		err = app.DB.UpdateOne(
			r.Context(),
			vlivesCollection,
			bson.M{"_id": id},
			bson.M{"$set": bson.M{"chat_enabled": enable}},
		)
		if err != nil {
			log.Printf("ChatEnable update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func ChatSlowMode(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		idHex := ps.ByName("liveid")
		id, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			writeError(w, "invalid liveid", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			writeError(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		stream, err := getStreamByID(r.Context(), app, id)
		if err != nil {
			writeError(w, "stream not found", http.StatusNotFound)
			return
		}
		if !isOwner(userID, stream) {
			writeError(w, "forbidden", http.StatusForbidden)
			return
		}

		var payload struct {
			Seconds int `json:"seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, "invalid body", http.StatusBadRequest)
			return
		}
		if payload.Seconds < 0 || payload.Seconds > 3600 {
			writeError(w, "invalid seconds", http.StatusBadRequest)
			return
		}

		err = app.DB.UpdateOne(
			r.Context(),
			vlivesCollection,
			bson.M{"_id": id},
			bson.M{"$set": bson.M{"slow_mode_seconds": payload.Seconds}},
		)
		if err != nil {
			log.Printf("ChatSlowMode update error: %v", err)
			writeError(w, "db error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
