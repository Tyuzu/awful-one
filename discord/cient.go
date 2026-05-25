package discord

import (
	"context"
	"naevis/infra/db"
	"naevis/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
)

type Client struct {
	/* Identity */
	ID     string
	UserID string
	RoomID string

	/* Transport */
	Conn *websocket.Conn
	Send chan Event

	/* Lifecycle */
	Ctx    context.Context
	Cancel context.CancelFunc

	/* Ordering / delivery */
	LastAckSeq int64 // last sequence acknowledged by client

	/* Flow control */
	mu        sync.Mutex
	connected bool

	/* Presence */
	JoinedAt time.Time
}

func (c *Client) WritePump() {
	defer c.Close()

	for {
		select {
		case evt, ok := <-c.Send:
			if !ok {
				return
			}

			c.mu.Lock()
			err := c.Conn.WriteJSON(evt)
			c.mu.Unlock()

			if err != nil {
				return
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

type ClientMessage struct {
	Type string `json:"type"`

	// ACK
	AckSeq int64 `json:"ackSeq,omitempty"`

	// typing
	Typing bool `json:"typing,omitempty"`
}

func (c *Client) ReadPump(hub *Hub) {
	defer c.Close()

	for {
		var msg ClientMessage
		if err := c.Conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "ACK":
			c.LastAckSeq = msg.AckSeq

		case "TYPING":
			hub.Emit(Event{
				Type:      TypingStart,
				RoomID:    c.RoomID,
				UserID:    c.UserID,
				Timestamp: time.Now(),
			})
		}
	}
}
func (c *Client) Close() {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return
	}
	c.connected = false
	c.mu.Unlock()

	c.Cancel()
	c.Conn.Close()
	close(c.Send)
}
func NewClient(
	conn *websocket.Conn,
	userID, roomID string,
) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		ID:        userID + ":" + roomID,
		UserID:    userID,
		RoomID:    roomID,
		Conn:      conn,
		Send:      make(chan Event, 128),
		Ctx:       ctx,
		Cancel:    cancel,
		connected: true,
		JoinedAt:  time.Now(),
	}
}
func (c *Client) CatchUp(
	ctx context.Context,
	db db.Database,
) error {
	if c.LastAckSeq == 0 {
		return nil
	}

	var msgs []models.Message
	err := db.FindMany(
		ctx,
		"messages",
		bson.M{
			"roomId": c.RoomID,
			"seq":    bson.M{"$gt": c.LastAckSeq},
		},
		&msgs,
	)

	if err != nil {
		return err
	}

	for _, m := range msgs {
		c.Send <- Event{
			Type:      MessageCreate,
			RoomID:    m.RoomID,
			UserID:    m.UserID,
			Payload:   m,
			Timestamp: m.CreatedAt,
		}
	}

	return nil
}
