package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

// WSHub manages WebSocket connections for a chat room
type WSHub struct {
	roomID     int64
	clients    map[*WSClient]bool
	mu         sync.RWMutex
	broadcast  chan *WSMessage
	register   chan *WSClient
	unregister chan *WSClient
	app        *App
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	hub      *WSHub
	conn     *websocket.Conn
	send     chan *WSMessage
	userID   int64
	username string
	roomID   int64
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string   `json:"type"` // "message", "join", "leave", "error"
	UserID    int64    `json:"user_id"`
	Username  string   `json:"username"`
	Content   string   `json:"content,omitempty"`
	Timestamp int64    `json:"timestamp"`
	Members   []Member `json:"members,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// Member represents a room member
type Member struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, validate origins properly
	},
}

// NewWSHub creates a new WebSocket hub for a chat room
func NewWSHub(roomID int64, app *App) *WSHub {
	hub := &WSHub{
		roomID:     roomID,
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan *WSMessage, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		app:        app,
	}

	go hub.run()
	return hub
}

// run manages the hub's event loop
func (h *WSHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			// Notify others of new member
			h.broadcast <- &WSMessage{
				Type:      "join",
				UserID:    client.userID,
				Username:  client.username,
				Timestamp: getCurrentTime(),
			}

			// Send member list to new client
			h.sendMemberList(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

			// Notify others of member leaving
			h.broadcast <- &WSMessage{
				Type:      "leave",
				UserID:    client.userID,
				Username:  client.username,
				Timestamp: getCurrentTime(),
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Channel full, skip message
					go func(c *WSClient) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// sendMemberList sends current member list to a client
func (h *WSHub) sendMemberList(client *WSClient) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	members := make([]Member, 0, len(h.clients))
	for c := range h.clients {
		members = append(members, Member{
			UserID:   c.userID,
			Username: c.username,
		})
	}

	msg := &WSMessage{
		Type:      "members",
		Members:   members,
		Timestamp: getCurrentTime(),
	}

	select {
	case client.send <- msg:
	default:
		log.Printf("failed to send member list to client %d", client.userID)
	}
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(getDeadlineTime(60))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(getDeadlineTime(60))
		return nil
	})

	for {
		var msg struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}

		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		if msg.Type == "message" {
			// Verify user is in room and save to persistent storage
			c.hub.app.roomMux.Lock()
			room, ok := c.hub.app.rooms[c.roomID]
			c.hub.app.roomMux.Unlock()

			if !ok || !room.Members[c.userID] {
				c.send <- &WSMessage{
					Type:      "error",
					Error:     "user not in room",
					Timestamp: getCurrentTime(),
				}
				continue
			}

			// Save message to persistent storage
			c.hub.app.roomMux.Lock()
			msgID := c.hub.app.nextMsgID
			c.hub.app.nextMsgID++
			message := Message{
				ID:        msgID,
				RoomID:    c.roomID,
				UserID:    c.userID,
				Content:   msg.Content,
				CreatedAt: time.Now(),
			}
			room.Messages = append(room.Messages, message)
			c.hub.app.roomMux.Unlock()

			// Broadcast to all connected clients
			c.hub.broadcast <- &WSMessage{
				Type:      "message",
				UserID:    c.userID,
				Username:  c.username,
				Content:   msg.Content,
				Timestamp: message.CreatedAt.Unix(),
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *WSClient) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		c.conn.SetWriteDeadline(getDeadlineTime(10))

		if err := c.conn.WriteJSON(message); err != nil {
			return
		}
	}

	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// ServeWS handles WebSocket connections
func (a *App) ServeWS(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Extract and verify JWT token
	token, err := ExtractTokenFromQuery(r)
	if err != nil {
		http.Error(w, "missing or invalid token", http.StatusUnauthorized)
		return
	}

	claims, err := a.jwtManager.Verify(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Extract room ID from URL
	roomID, err := parseID(ps.ByName("id"))
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	// Verify room exists and user is member
	a.roomMux.RLock()
	room, ok := a.rooms[roomID]
	a.roomMux.RUnlock()

	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	if !room.Members[claims.UserID] {
		http.Error(w, "user not in room", http.StatusForbidden)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	// Get or create hub for this room
	a.hubsMu.Lock()
	hub, exists := a.hubs[roomID]
	if !exists {
		hub = NewWSHub(roomID, a)
		a.hubs[roomID] = hub
	}
	a.hubsMu.Unlock()

	// Create client and register
	client := &WSClient{
		hub:      hub,
		conn:     conn,
		send:     make(chan *WSMessage, 256),
		userID:   claims.UserID,
		username: claims.Username,
		roomID:   roomID,
	}

	hub.register <- client

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}

// Helper functions

func getCurrentTime() int64 {
	return time.Now().Unix()
}

func getDeadlineTime(offsetSeconds int) time.Time {
	if offsetSeconds == 0 {
		return time.Now()
	}
	return time.Now().Add(time.Duration(offsetSeconds) * time.Second)
}
