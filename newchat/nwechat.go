package newchat

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/utils"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// ------------------------- Helpers -------------------------

func closeChanSafe(ch chan []byte) {
	defer func() { _ = recover() }()
	close(ch)
}

func originAllowed(app *infra.Deps) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow requests without Origin header
		}
		// Check if origin is in allowed list
		for _, allowed := range app.Config.AllowedOrigins {
			if origin == allowed {
				return true
			}
		}
		log.Printf("WebSocket connection rejected from unauthorized origin: %s", origin)
		return false
	}
}

// ------------------------- Hub lifecycle -------------------------

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastMsg),
		stopChan:   make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.stopChan:
			return
		case c := <-h.register:
			h.mu.Lock()
			if h.stopped {
				h.mu.Unlock()
				c.cancel()
				c.Conn.Close()
				closeChanSafe(c.Send)
				continue
			}
			if h.rooms[c.Room] == nil {
				h.rooms[c.Room] = make(map[*Client]bool)
			}
			h.rooms[c.Room][c] = true
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if clients := h.rooms[c.Room]; clients != nil {
				if _, ok := clients[c]; ok {
					delete(clients, c)
					closeChanSafe(c.Send)
				}
				if len(clients) == 0 {
					delete(h.rooms, c.Room)
				}
			}
			h.mu.Unlock()
		case m := <-h.broadcast:
			h.mu.Lock()
			if clients := h.rooms[m.Room]; clients != nil {
				// Create a copy of clients to iterate over (avoid map modification during iteration)
				sclientsList := make([]*Client, 0, len(clients))
				for client := range clients {
					sclientsList = append(sclientsList, client)
				}
				h.mu.Unlock()

				for _, client := range sclientsList {
					select {
					case client.Send <- m.Data:
					default:
						closeChanSafe(client.Send)
						h.mu.Lock()
						delete(clients, client)
						h.mu.Unlock()
					}
				}
			} else {
				h.mu.Unlock()
			}
		}
	}
}

func (h *Hub) Stop() {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return
	}
	h.stopped = true
	h.stopOnce.Do(func() { close(h.stopChan) })

	roomsCopy := h.rooms
	h.rooms = make(map[string]map[*Client]bool)
	h.mu.Unlock()

	for _, clients := range roomsCopy {
		for c := range clients {
			c.cancel()
			c.Conn.Close()
			closeChanSafe(c.Send)
		}
	}
	log.Println("hub stopped")
}

// ------------------------- WebSocket -------------------------

func WebSocketHandler(hub *Hub, app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		room := strings.TrimSpace(ps.ByName("room"))
		if room == "" {
			http.Error(w, "room required", http.StatusBadRequest)
			return
		}

		var token string
		if auth := r.Header.Get("Authorization"); auth != "" {
			token = auth
		} else if q := r.URL.Query().Get("token"); q != "" {
			token = "Bearer " + q
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := utils.ValidateJWT(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		upgraderWithOrigin := websocket.Upgrader{
			CheckOrigin: originAllowed(app),
		}
		conn, err := upgraderWithOrigin.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		client := &Client{
			Conn:   conn,
			Send:   make(chan []byte, 256),
			Room:   room,
			UserID: claims.UserID,
			ctx:    ctx,
			cancel: cancel,
		}

		// send history
		go func() {
			opts := db.FindManyOptions{
				Sort:  bson.D{{Key: "timestamp", Value: -1}},
				Limit: 20,
			}

			var history []Message
			if err := app.DB.FindManyWithOptions(ctx, messagesCollection, map[string]any{"room": room}, opts, &history); err != nil {
				return
			}

			for i := len(history) - 1; i >= 0; i-- {
				out := outboundPayload{
					Action:    "chat",
					ID:        history[i].MessageID,
					Room:      history[i].Room,
					SenderID:  history[i].SenderID,
					Content:   history[i].Content,
					Files:     history[i].Files,
					Timestamp: history[i].Timestamp,
				}
				if data, err := json.Marshal(out); err == nil {
					select {
					case client.Send <- data:
					case <-client.ctx.Done():
						return
					}
				}
			}
		}()

		hub.register <- client
		go writePump(client, hub)
		go readPump(client, hub, app)
	}
}

func writePump(c *Client, hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.cancel()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

func readPump(c *Client, hub *Hub, app *infra.Deps) {
	defer func() {
		hub.unregister <- c
		c.cancel()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, raw, err := c.Conn.ReadMessage()
			if err != nil {
				return
			}

			var in inboundPayload
			if err := json.Unmarshal(raw, &in); err != nil {
				continue
			}

			switch in.Action {
			case "chat":
				msg := Message{
					MessageID: utils.GenerateRandomDigitString(16),
					Room:      c.Room,
					SenderID:  c.UserID,
					Content:   in.Content,
					Timestamp: time.Now().Unix(),
				}

				if err := app.DB.InsertOne(c.ctx, messagesCollection, msg); err != nil {
					continue
				}

				out := outboundPayload{
					Action:    "chat",
					ID:        msg.MessageID,
					Room:      msg.Room,
					SenderID:  msg.SenderID,
					Content:   msg.Content,
					Timestamp: msg.Timestamp,
				}
				data, _ := json.Marshal(out)
				hub.broadcast <- broadcastMsg{Room: c.Room, Data: data}

			case "edit":
				if err := UpdatexMessage(c.UserID, in.ID, in.Content, app); err != nil {
					continue
				}
				out := outboundPayload{
					Action:    "edit",
					ID:        in.ID,
					Content:   in.Content,
					Timestamp: time.Now().Unix(),
				}
				data, _ := json.Marshal(out)
				hub.broadcast <- broadcastMsg{Room: c.Room, Data: data}

			case "delete":
				if err := DeletexMessage(c.UserID, in.ID, app); err != nil {
					continue
				}
				out := outboundPayload{
					Action: "delete",
					ID:     in.ID,
				}
				data, _ := json.Marshal(out)
				hub.broadcast <- broadcastMsg{Room: c.Room, Data: data}
			}
		}
	}
}
