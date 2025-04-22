package events

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

type Events interface {
	Publish(event LifecycleEvent, payload ...any) error
	Subscribe(event LifecycleEvent) (<-chan *message.Message, error)
}

type events struct {
	channel *gochannel.GoChannel
}

func New() Events {
	channel := gochannel.NewGoChannel(
		gochannel.Config{},
		watermill.NopLogger{},
	)

	return &events{
		channel: channel,
	}
}

func (e *events) Publish(event LifecycleEvent, payload ...any) error {
	var messages []*message.Message
	for _, p := range payload {
		pb, err := json.Marshal(p)
		if err != nil {
			return err
		}

		messages = append(messages, message.NewMessage(watermill.NewUUID(), pb))
	}

	return e.channel.Publish(string(event), messages...)
}

func (e *events) Subscribe(event LifecycleEvent) (<-chan *message.Message, error) {
	return e.channel.Subscribe(context.Background(), string(event))
}
