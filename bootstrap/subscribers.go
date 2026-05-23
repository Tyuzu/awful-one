package bootstrap

import (
	"context"

	authsub "naevis/infra/mq/subscriber/auth"
	ordersub "naevis/infra/mq/subscriber/order"
	refundsub "naevis/infra/mq/subscriber/refund"

	"naevis/infra"
	"naevis/infra/mq/subscriber"
)

func RegisterSubscribers(
	ctx context.Context,
	app *infra.Deps,
) error {

	subscribers := []subscriber.Subscriber{
		authsub.New(),
		ordersub.New(),
		refundsub.New(),
	}

	for _, sub := range subscribers {
		if err := sub.Register(ctx, app.MQ); err != nil {
			return err
		}
	}

	return nil
}
