package models

import (
	"time"
)

// Notification represents a user-level notification
type Notification struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	UserID      string    `bson:"userId" json:"userId"`
	Type        string    `bson:"type" json:"type"`               // notification type: mention, follow, comment, like, etc.
	Title       string    `bson:"title" json:"title"`             // notification title
	Message     string    `bson:"message" json:"message"`         // notification message/content
	EntityType  string    `bson:"entityType" json:"entityType"`   // type of entity this notification is about
	EntityID    string    `bson:"entityId" json:"entityId"`       // ID of the entity
	RelatedUser string    `bson:"relatedUser" json:"relatedUser"` // user who triggered this notification
	IsRead      bool      `bson:"isRead" json:"isRead"`           // read status
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt" json:"updatedAt"`
}

// NotificationPreference stores notification settings per user
type NotificationPreference struct {
	ID              string    `bson:"_id,omitempty" json:"id"`
	UserID          string    `bson:"userId" json:"userId"`
	MentionsEnabled bool      `bson:"mentionsEnabled" json:"mentionsEnabled"`
	FollowsEnabled  bool      `bson:"followsEnabled" json:"followsEnabled"`
	CommentsEnabled bool      `bson:"commentsEnabled" json:"commentsEnabled"`
	LikesEnabled    bool      `bson:"likesEnabled" json:"likesEnabled"`
	MessagesEnabled bool      `bson:"messagesEnabled" json:"messagesEnabled"`
	AllEnabled      bool      `bson:"allEnabled" json:"allEnabled"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time `bson:"updatedAt" json:"updatedAt"`
}
