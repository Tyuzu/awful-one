package auth

import (
	"context"

	"naevis/infra"
	"naevis/infra/mq"
	"naevis/infra/mq/subscriber"
)

// Subscriber registers all auth domain event handlers
// Implements the mq.subscriber.Subscriber interface
type Subscriber struct {
	deps *infra.Deps
}

// NewSubscriber creates a new auth domain subscriber
func NewSubscriber(deps *infra.Deps) subscriber.Subscriber {
	return &Subscriber{
		deps: deps,
	}
}

// Register subscribes to all auth-related domain events
func (s *Subscriber) Register(ctx context.Context, bus mq.MQ) error {
	// TODO: Implement proper UserRepository interface in infra/db layer
	// For now, this is a placeholder that needs integration with actual DB layer
	// handler := NewEventHandler(userRepository)

	// Placeholder: subscribe to events when repository is available
	// if err := bus.QueueSubscribe(...) { ... }

	return nil
}
