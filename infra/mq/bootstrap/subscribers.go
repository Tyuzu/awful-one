package bootstrap

import (
	"context"

	authsub "naevis/domain/auth"
	mediasub "naevis/domain/media"
	ordersub "naevis/domain/order"

	"naevis/infra"
	"naevis/infra/mq/subscriber"
)

// RegisterSubscribers registers all domain event subscribers
// Each domain owns its event subscriptions and implements the Subscriber interface
func RegisterSubscribers(
	ctx context.Context,
	app *infra.Deps,
) error {

	subscribers := []subscriber.Subscriber{
		authsub.NewSubscriber(app),
		mediasub.NewSubscriber(app),
		ordersub.NewSubscriber(app),
		// Additional domains can be easily added here without modifying bootstrap logic
	}

	for _, sub := range subscribers {
		if err := sub.Register(ctx, app.MQ); err != nil {
			return err
		}
	}

	return nil
}
