package order

import (
	"context"

	"naevis/infra"
	"naevis/infra/mq"
	"naevis/infra/mq/subscriber"
)

// Subscriber registers all order domain event handlers
// Implements the mq.subscriber.Subscriber interface
type Subscriber struct {
	deps *infra.Deps
}

// NewSubscriber creates a new order domain subscriber
func NewSubscriber(deps *infra.Deps) subscriber.Subscriber {
	return &Subscriber{
		deps: deps,
	}
}

// Register subscribes to all order-related domain events
func (s *Subscriber) Register(ctx context.Context, bus mq.MQ) error {
	// TODO: Implement proper OrderRepository interface in infra/db layer
	// For now, this is a placeholder that needs integration with actual DB layer
	// handler := NewEventHandler(orderRepository)

	// Placeholder: subscribe to events when repository is available
	// if err := bus.QueueSubscribe(...) { ... }

	return nil
}
