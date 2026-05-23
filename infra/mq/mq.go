package mq

import "context"

type MessageHandler func(ctx context.Context, data []byte) error

type MQ interface {
	Emit(ctx context.Context, subject string, data []byte) error
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler MessageHandler) error
	QueueSubscribe(ctx context.Context, subject string, queue string, handler MessageHandler) error
}
