package models

import (
	"time"
)

// Correct Notice model (in case you define here)
type Notice struct {
	NoticeID   string    `bson:"noticeid,omitempty" json:"noticeid"`
	EntityType string    `bson:"entityType" json:"entityType"`
	EntityId   string    `bson:"entityId" json:"entityId"`
	Title      string    `bson:"title" json:"title"`
	Content    string    `bson:"content,omitempty" json:"content,omitempty"`
	Summary    string    `bson:"summary" json:"summary"`
	CreatedBy  string    `bson:"createdBy" json:"createdBy"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt"`
}
