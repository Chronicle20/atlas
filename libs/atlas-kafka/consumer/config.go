package consumer

import (
	"encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"time"
)

type HandlerFunc[E any] func(logrus.FieldLogger, opentracing.Span, E)

type messageHandler func(l logrus.FieldLogger, span opentracing.Span, msg kafka.Message)

func NewConfig[E any](name string, topic string, groupId string, handler HandlerFunc[E]) Config {
	return Config{
		name:    name,
		topic:   topic,
		groupId: groupId,
		maxWait: 500 * time.Millisecond,
		handler: adapt(handler),
	}
}

type Config struct {
	name    string
	topic   string
	groupId string
	maxWait time.Duration
	handler messageHandler
}

func adapt[E any](eh HandlerFunc[E]) messageHandler {
	return func(l logrus.FieldLogger, span opentracing.Span, msg kafka.Message) {
		var event E
		err := json.Unmarshal(msg.Value, &event)
		if err != nil {
			l.WithError(err).Errorf("Could not unmarshal event into %s.", msg.Value)
		} else {
			eh(l, span, event)
		}
	}
}
