package mq

import (
	"context"

	"github.com/nats-io/nats.go"
)

type JetStreamMQ struct {
	js       nats.JetStreamContext
	consumer string
}

func NewJetStreamMQ(js nats.JetStreamContext, consumer string) *JetStreamMQ {
	return &JetStreamMQ{
		js:       js,
		consumer: consumer,
	}
}

func (j *JetStreamMQ) Emit(ctx context.Context, subject string, data []byte) error {
	_, err := j.js.Publish(subject, data)
	return err
}

func (j *JetStreamMQ) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := j.js.Publish(subject, data)
	return err
}

func (j *JetStreamMQ) Subscribe(
	ctx context.Context,
	subject string,
	handler MessageHandler,
) error {
	sub, err := j.js.Subscribe(
		subject,
		func(msg *nats.Msg) {
			if err := handler(ctx, msg.Data); err != nil {
				_ = msg.Nak()
				return
			}
			_ = msg.Ack()
		},
		nats.Durable(j.consumer),
		nats.ManualAck(),
		nats.AckExplicit(),
	)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()

	return nil
}

func (j *JetStreamMQ) QueueSubscribe(
	ctx context.Context,
	subject string,
	queue string,
	handler MessageHandler,
) error {

	sub, err := j.js.QueueSubscribe(
		subject,
		queue,
		func(msg *nats.Msg) {
			if err := handler(ctx, msg.Data); err != nil {
				_ = msg.Nak()
				return
			}

			_ = msg.Ack()
		},
		nats.Durable(queue),
		nats.ManualAck(),
		nats.AckExplicit(),
	)

	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()

	return nil
}
