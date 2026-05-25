package models

import (
	"time"
)

/*
Livestream state machine
*/
const (
	LiveCreated   = "CREATED"
	LiveReady     = "READY"
	LiveLive      = "LIVE"
	LiveEnded     = "ENDED"
	LiveCancelled = "CANCELLED"
	LiveError     = "ERROR"
)

/*
LiveStream represents a single livestream lifecycle.
*/
type LiveStream struct {
	LiveID string `bson:"liveid,omitempty" json:"liveid"`

	// Ownership / scope
	EntityType string `bson:"entity_type" json:"entityType"`
	EntityID   string `bson:"entity_id" json:"entityId"`
	CreatorID  string `bson:"creator_id" json:"creatorId"`

	// Metadata
	Title       string   `bson:"title" json:"title"`
	Description string   `bson:"description,omitempty" json:"description,omitempty"`
	Thumbnail   string   `bson:"thumbnail,omitempty" json:"thumbnail,omitempty"`
	Tags        []string `bson:"tags,omitempty" json:"tags,omitempty"`
	State       string   `bson:"state" json:"state"`

	// Visibility
	IsPublic bool `bson:"is_public" json:"isPublic"`
	Unlisted bool `bson:"unlisted" json:"unlisted"`

	// Streaming / ingest (NEVER expose)
	StreamKey string `bson:"stream_key" json:"-"`
	IngestURL string `bson:"ingest_url" json:"-"`

	// Playback
	PlaybackURL string `bson:"playback_url,omitempty" json:"playbackUrl,omitempty"`

	// Recording / VOD (internal paths hidden)
	RecordingPath string `bson:"recording_path,omitempty" json:"-"`
	VODURL        string `bson:"vod_url,omitempty" json:"vodUrl,omitempty"`
	VODPublished  bool   `bson:"vod_published" json:"vodPublished"`

	// Chat
	ChatEnabled     bool `bson:"chat_enabled" json:"chatEnabled"`
	SlowModeSeconds int  `bson:"slow_mode_seconds" json:"slowModeSeconds"`

	// Scheduling / timing
	ScheduledAt time.Time `bson:"scheduled_at,omitempty" json:"scheduledAt,omitempty"`
	ReadyAt     time.Time `bson:"ready_at,omitempty" json:"readyAt,omitempty"`
	StartedAt   time.Time `bson:"started_at,omitempty" json:"startedAt,omitempty"`
	EndedAt     time.Time `bson:"ended_at,omitempty" json:"endedAt,omitempty"`

	// Audit
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at,omitempty" json:"updatedAt,omitempty"` // optional for tracking edits

	// Runtime-only fields (not persisted)
	ViewerCount int `bson:"-" json:"viewerCount,omitempty"`
}
