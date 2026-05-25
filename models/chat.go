package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	MessageID  string     `bson:"messageid,omitempty"        json:"messageid"`
	ChatID     string     `bson:"chatid"              json:"chatid"`
	RoomID     string     `bson:"roomid"              json:"roomid"`
	UserID     string     `bson:"userid"              json:"userid"`
	Text       string     `bson:"text,omitempty" json:"text,omitempty"`
	FileURL    string     `bson:"fileURL,omitempty" json:"fileURL,omitempty"`
	FileType   string     `bson:"fileType,omitempty" json:"fileType,omitempty"` // "image" or "video"
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt"`
	ReplyTo    *ReplyRef  `bson:"replyTo,omitempty" json:"replyTo,omitempty"`
	SenderName string     `bson:"senderName,omitempty" json:"senderName,omitempty"`
	AvatarURL  string     `bson:"avatarUrl,omitempty"   json:"avatarUrl,omitempty"`
	Content    string     `bson:"content"           json:"content"`
	Media      *Media     `bson:"media,omitempty"   json:"media,omitempty"`
	EditedAt   *time.Time `bson:"editedAt,omitempty" json:"editedAt,omitempty"`
	Deleted    bool       `bson:"deleted"           json:"deleted"`
	ReadBy     []string   `bson:"readBy,omitempty"  json:"readBy,omitempty"`
	Status     string     `bson:"status,omitempty"  json:"status,omitempty"` // e.g. "sent", "read"
	Nonce      string     `bson:"nonce" json:"nonce"`
	Seq        int64      `bson:"seq" json:"seq"`
}

type Chat struct {
	Users        []string        `bson:"users" json:"users"`
	LastMessage  MessagePreview  `bson:"lastMessage" json:"lastMessage"`
	ReadStatus   map[string]bool `bson:"readStatus,omitempty" json:"readStatus,omitempty"`
	ChatID       string          `bson:"chatid,omitempty" json:"chatid"`
	Participants []string        `bson:"participants"      json:"participants"`
	CreatedAt    time.Time       `bson:"createdAt"         json:"createdAt"`
	UpdatedAt    time.Time       `bson:"updatedAt"         json:"updatedAt"`
	EntityType   string          `bson:"entitytype"        json:"entitytype"`
	EntityId     string          `bson:"entityid"          json:"entityid"`
}

type MessagePreview struct {
	Text      string    `bson:"text" json:"text"`
	UserID    string    `bson:"userid" json:"userid"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

// ReplyRef represents the clientâ€side â€œreplyToâ€ payload.
type ReplyRef struct {
	ID   string `json:"id"`
	User string `json:"user"`
	Text string `json:"text"`
}

type Like struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     string             `bson:"userid"`
	EntityType string             `bson:"entity_type"` // e.g. "post"
	EntityID   string             `bson:"entity_id"`   // e.g. post ID
	CreatedAt  time.Time          `bson:"created_at"`
}

// IncomingWSMessage represents a generic WebSocket inbound payload
type IncomingWSMessage struct {
	Type      string `json:"type"`
	ChatID    string `json:"chatid"`
	Content   string `json:"content"`
	MediaURL  string `json:"mediaUrl"`
	MediaType string `json:"mediaType"`
	Online    bool   `json:"online"`
	ClientID  string `json:"clientId,omitempty"`
}

// Media represents media attached to a message
type MessageMedia struct {
	URL  string `bson:"url"  json:"url"`
	Type string `bson:"type" json:"type"`
}
