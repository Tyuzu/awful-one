package discord

import (
	"context"
	"encoding/json"
	"naevis/infra/mq"
	"naevis/models"
)

func StartConsumer(ctx context.Context, mq mq.MQ, hub *Hub, subject string) error {
	return mq.Subscribe(ctx, subject, func(ctx context.Context, data []byte) error {
		var msg models.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}

		hub.Emit(Event{
			Type:      MessageCreate,
			RoomID:    msg.RoomID,
			UserID:    msg.UserID,
			Payload:   msg,
			Timestamp: msg.CreatedAt,
		})

		return nil
	})
}
