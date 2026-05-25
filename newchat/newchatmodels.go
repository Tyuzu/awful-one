package newchat

import (
	"context"
	"naevis/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ------------------------- Types -------------------------

type Hub struct {
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastMsg

	mu       sync.Mutex
	stopped  bool
	stopChan chan struct{}
	stopOnce sync.Once
}

type Client struct {
	Conn   *websocket.Conn
	Send   chan []byte
	Room   string
	UserID string

	ctx    context.Context
	cancel context.CancelFunc
}

type Attachment struct {
	Filename string `bson:"filename" json:"filename"`
	Path     string `bson:"path" json:"path"`
}

type Message struct {
	ChatID     string           `bson:"chatid"              json:"chatid"`
	UserID     string           `bson:"sender"              json:"sender"`
	Text       string           `bson:"text,omitempty" json:"text,omitempty"`
	FileURL    string           `bson:"fileURL,omitempty" json:"fileURL,omitempty"`
	FileType   string           `bson:"fileType,omitempty" json:"fileType,omitempty"` // "image" or "video"
	CreatedAt  time.Time        `bson:"createdAt" json:"createdAt"`
	ReplyTo    *models.ReplyRef `bson:"replyTo,omitempty" json:"replyTo,omitempty"`
	SenderName string           `bson:"senderName,omitempty" json:"senderName,omitempty"`
	AvatarURL  string           `bson:"avatarUrl,omitempty"   json:"avatarUrl,omitempty"`
	Media      *models.Media    `bson:"media,omitempty"   json:"media,omitempty"`
	EditedAt   *time.Time       `bson:"editedAt,omitempty" json:"editedAt,omitempty"`
	Deleted    bool             `bson:"deleted"           json:"deleted"`
	ReadBy     []string         `bson:"readBy,omitempty"  json:"readBy,omitempty"`
	Status     string           `bson:"status,omitempty"  json:"status,omitempty"` // e.g. "sent", "read"

	MessageID string       `bson:"messageid" json:"messageid"`
	Room      string       `bson:"room" json:"room"`
	SenderID  string       `bson:"senderid" json:"senderid"`
	Content   string       `bson:"content" json:"content"`
	Files     []Attachment `bson:"files,omitempty" json:"files,omitempty"`
	Timestamp int64        `bson:"timestamp" json:"timestamp"`
}

type inboundPayload struct {
	Action  string `json:"action"`
	ID      string `json:"id,omitempty"`
	Content string `json:"content,omitempty"`
}

type outboundPayload struct {
	Action    string       `json:"action"`
	ID        string       `json:"id,omitempty"`
	Room      string       `json:"room,omitempty"`
	SenderID  string       `json:"senderid,omitempty"`
	Content   string       `json:"content,omitempty"`
	Files     []Attachment `json:"files,omitempty"`
	Timestamp int64        `json:"timestamp,omitempty"`
}

type broadcastMsg struct {
	Room string
	Data []byte
}

// type searchResult struct {
// 	Matches []ChatMessage `json:"matches"`
// }

// type chatMessage struct {
// 	ID        string `json:"id"`
// 	Sender    string `json:"sender"`
// 	Text      string `json:"text"`
// 	Timestamp string `json:"timestamp"`
// }
