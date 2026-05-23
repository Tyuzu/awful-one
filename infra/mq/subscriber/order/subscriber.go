package order

import (
	"context"

	"naevis/infra/mq"
)

type Subscriber struct{}

func New() *Subscriber {
	return &Subscriber{}
}

func (s *Subscriber) Register(
	ctx context.Context,
	bus mq.MQ,
) error {

	return nil
}
