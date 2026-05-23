package mechat

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

//
// ================= HUB =================
//

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan any
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func (h *Hub) add(c *Client) {
	h.mu.Lock()
	h.clients[c.UserID] = c
	h.mu.Unlock()
}

func (h *Hub) remove(user string) {
	h.mu.Lock()
	delete(h.clients, user)
	h.mu.Unlock()
}

//
// ================= WS SETUP =================
//

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	writeTimeout = 10 * time.Second
	pongWait     = 60 * time.Second
	pingPeriod   = 30 * time.Second
	queueSize    = 256
)

func HandleWebSocket(app *infra.Deps, hub *Hub) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		raw := r.URL.Query().Get("token")
		if raw == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := utils.ParseToken(raw)
		if err != nil || claims.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "upgrade failed", http.StatusInternalServerError)
			return
		}

		client := &Client{
			UserID: claims.UserID,
			Conn:   conn,
			Send:   make(chan any, queueSize),
		}
		hub.add(client)

		defer func() {
			hub.remove(client.UserID)
			close(client.Send)
			_ = conn.Close()
		}()

		go wsWriter(client)
		go wsPing(client)
		wsReader(r.Context(), client, app, hub)
	}
}

//
// ================= LOOPS =================
//

func wsWriter(c *Client) {
	for msg := range c.Send {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := c.Conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

func wsPing(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := c.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout)); err != nil {
			return
		}
	}
}

func wsReader(ctx context.Context, c *Client, app *infra.Deps, hub *Hub) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var in models.IncomingWSMessage
		if err := c.Conn.ReadJSON(&in); err != nil {
			return
		}

		switch in.Type {
		case "message":
			wsSendMessage(ctx, c, in, app, hub)
		case "edit":
			wsEditMessage(ctx, c, in, app, hub)
		case "delete":
			wsDeleteMessage(ctx, c, in, app, hub)
		case "read":
			wsReadMessage(ctx, c, in, app, hub)
		case "reaction:add":
			wsReaction(ctx, c, in, app, hub, true)
		case "reaction:remove":
			wsReaction(ctx, c, in, app, hub, false)
		}
	}
}

//
// ================= ACTIONS =================
//

func wsSendMessage(ctx context.Context, c *Client, in models.IncomingWSMessage, app *infra.Deps, hub *Hub) {
	if ensureChatAccess(ctx, app, in.ChatID, c.UserID) != nil {
		return
	}

	msg := &models.Message{
		MessageID: utils.GenerateRandomDigitString(16),
		ChatID:    in.ChatID,
		UserID:    c.UserID,
		Content:   in.Content,
		CreatedAt: time.Now(),
		ReadBy:    []string{c.UserID},
		Status:    "sent",
	}

	if err := app.DB.InsertOne(ctx, MessagesCollection, msg); err != nil {
		log.Println("insert failed:", err)
		return
	}

	updateLastMessage(ctx, app, in.ChatID, msg)

	broadcastToChat(ctx, app, hub, in.ChatID, map[string]any{
		"type":      "message",
		"messageid": msg.MessageID,
		"chatid":    msg.ChatID,
		"sender":    msg.UserID,
		"content":   msg.Content,
		"createdAt": msg.CreatedAt,
		"status":    "sent",
		"clientId":  in.ClientID,
	})
}

func wsEditMessage(ctx context.Context, c *Client, in models.IncomingWSMessage, app *infra.Deps, hub *Hub) {
	msgID := in.Content
	if msgID == "" {
		return
	}

	now := time.Now()
	var msg models.Message
	if err := app.DB.FindOneAndUpdate(
		ctx,
		MessagesCollection,
		map[string]any{
			"messageid":  msgID,
			"userid":     c.UserID,
			"deleted_ne": true,
		},
		map[string]any{
			"content":  in.MediaURL,
			"editedAt": &now,
		},
		&msg,
	); err != nil {
		return
	}

	updateLastMessage(ctx, app, msg.ChatID, &msg)

	broadcastToChat(ctx, app, hub, msg.ChatID, map[string]any{
		"type":    "edit",
		"message": msg,
	})
}

func wsDeleteMessage(ctx context.Context, c *Client, in models.IncomingWSMessage, app *infra.Deps, hub *Hub) {
	msgID := in.Content
	if msgID == "" {
		return
	}

	var msg models.Message
	if err := app.DB.FindOneAndUpdate(
		ctx,
		MessagesCollection,
		map[string]any{
			"messageid": msgID,
			"userid":    c.UserID,
		},
		map[string]any{"deleted": true},
		&msg,
	); err != nil {
		return
	}

	_ = app.DB.UpdateOne(
		ctx,
		MereChatCollection,
		map[string]any{
			"chatid":               msg.ChatID,
			"lastMessage.senderId": msg.UserID,
		},
		map[string]any{"lastMessage": nil},
	)

	broadcastToChat(ctx, app, hub, msg.ChatID, map[string]any{
		"type":      "delete",
		"messageId": msg.MessageID,
	})
}

func wsReadMessage(ctx context.Context, c *Client, in models.IncomingWSMessage, app *infra.Deps, hub *Hub) {
	msgID := in.Content
	if msgID == "" {
		return
	}

	if err := app.DB.AddToSet(
		ctx,
		MessagesCollection,
		map[string]any{"messageid": msgID},
		"readBy",
		c.UserID,
	); err != nil {
		return
	}

	broadcastToChat(ctx, app, hub, in.ChatID, map[string]any{
		"type":      "read",
		"messageId": msgID,
		"user":      c.UserID,
	})
}

func wsReaction(ctx context.Context, c *Client, in models.IncomingWSMessage, app *infra.Deps, hub *Hub, add bool) {
	msgID := in.Content
	if msgID == "" {
		return
	}

	if add {
		_ = app.DB.AddToSet(ctx, MessagesCollection, map[string]any{"messageid": msgID}, "reactions", c.UserID)
	} else {
		_ = app.DB.UpdateOne(
			ctx,
			MessagesCollection,
			map[string]any{"messageid": msgID},
			map[string]any{"reactions": []string{}},
		)
	}

	broadcastToChat(ctx, app, hub, in.ChatID, map[string]any{
		"type":      "reaction",
		"messageId": msgID,
		"user":      c.UserID,
		"add":       add,
	})
}

//
// ================= BROADCAST =================
//

func broadcastToChat(ctx context.Context, app *infra.Deps, hub *Hub, chatID string, payload any) {
	var chat models.Chat
	if err := app.DB.FindOne(ctx, MereChatCollection, map[string]any{"chatid": chatID}, &chat); err != nil {
		return
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for _, uid := range chat.Participants {
		if c, ok := hub.clients[uid]; ok {
			select {
			case c.Send <- payload:
			default:
				log.Printf("ws drop to %s", uid)
			}
		}
	}
}
