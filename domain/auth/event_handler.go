package auth

import (
	"context"
	"encoding/json"

	"naevis/config/mqevent"
)

// EventHandler handles all authentication domain events
type EventHandler struct {
	userRepo UserRepository
}

// NewEventHandler creates a new authentication event handler
func NewEventHandler(userRepo UserRepository) *EventHandler {
	return &EventHandler{
		userRepo: userRepo,
	}
}

// HandleUserRegistered processes UserRegistered domain event
// This runs when a user completes registration
func (h *EventHandler) HandleUserRegistered(ctx context.Context, payload []byte) error {
	var event mqevent.UserRegisteredPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Example: Mark user as registered in a separate collection for analytics
	// or trigger welcome email, etc.
	_ = event // Use event data as needed

	return nil
}

// HandleUserLoggedIn processes UserLoggedIn domain event
// This runs when a user successfully logs in
func (h *EventHandler) HandleUserLoggedIn(ctx context.Context, payload []byte) error {
	var event mqevent.UserLoggedInPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Example: Update user's last login timestamp
	// Example: Log login event for security auditing
	// Example: Update user activity metrics
	_ = event // Use event data as needed

	return nil
}

// HandleUserLoggedOut processes UserLoggedOut domain event
// This runs when a user logs out
func (h *EventHandler) HandleUserLoggedOut(ctx context.Context, payload []byte) error {
	var event mqevent.UserLoggedOutPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Example: Invalidate user sessions
	// Example: Log logout event
	// Example: Trigger cleanup operations
	_ = event // Use event data as needed

	return nil
}
