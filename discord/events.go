package discord

import "time"

type EventType string

const (
	MessageCreate EventType = "MESSAGE_CREATE"
	MessageUpdate EventType = "MESSAGE_UPDATE"
	MessageDelete EventType = "MESSAGE_DELETE"

	TypingStart EventType = "TYPING_START"
	Presence    EventType = "PRESENCE_UPDATE"
	RoleUpdate  EventType = "ROLE_UPDATE"
)

type Event struct {
	Type      EventType `json:"type"`
	RoomID    string    `json:"roomId"`
	UserID    string    `json:"userId"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"ts"`
}
