package discord

import (
	"context"
	"encoding/json"
	"naevis/infra/cache"
	"naevis/infra/db"
	"naevis/infra/mq"
	"naevis/models"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func CreateMessage(
	ctx context.Context,
	db db.Database,
	cache cache.Cache,
	mq mq.MQ,
	room, user, content, nonce string,
) (*models.Message, error) {

	key := "msg:nonce:" + room + ":" + nonce
	ok, err := cache.SetNX(ctx, key, []byte("1"), 10*time.Minute)
	if err != nil {
		return nil, err
	}

	if !ok {
		var existing models.Message
		db.FindOne(ctx, "messages", bson.M{
			"roomId": room,
			"nonce":  nonce,
		}, &existing)
		return &existing, nil
	}

	var msg models.Message

	err = db.RunTransaction(ctx, func(tx context.Context) error {
		seq, err := cache.Incr(tx, "room:"+room+":seq")
		if err != nil {
			return err
		}

		msg = models.Message{
			MessageID: room + ":" + strconv.FormatInt(seq, 10),
			RoomID:    room,
			UserID:    user,
			Content:   content,
			Seq:       seq,
			Nonce:     nonce,
			CreatedAt: time.Now(),
		}

		return db.Insert(tx, "messages", msg)
	})

	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(msg)
	mq.Publish(ctx, "chat."+room+".events", b)

	return &msg, nil
}
