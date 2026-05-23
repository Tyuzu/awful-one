package subscriber

import (
	"context"

	"naevis/infra/mq"
)

type Subscriber interface {
	Register(
		ctx context.Context,
		bus mq.MQ,
	) error
}
