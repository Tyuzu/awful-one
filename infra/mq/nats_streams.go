package mq

import "github.com/nats-io/nats.go"

func EnsureStreams(js nats.JetStreamContext) error {
	streams := []*nats.StreamConfig{
		{
			Name: "AUTH",
			Subjects: []string{
				"auth.>",
			},
		},
		{
			Name: "ORDER",
			Subjects: []string{
				"order.>",
			},
		},
		{
			Name: "CHAT",
			Subjects: []string{
				"chat.>",
			},
		},
		{
			Name: "MEDIA",
			Subjects: []string{
				"media.>",
			},
		},
	}

	for _, stream := range streams {
		_, err := js.StreamInfo(stream.Name)

		if err == nil {
			continue
		}

		if _, err := js.AddStream(stream); err != nil {
			return err
		}
	}

	return nil
}
