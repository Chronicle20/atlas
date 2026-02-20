package _map

import (
	consumer2 "atlas-transports/kafka/consumer"
	"atlas-transports/instance"
	_map2 "atlas-transports/kafka/message/map"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(_map2.EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(_map2.EnvEventTopicMapStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterEnter))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterExit))); err != nil {
			return err
		}
		return nil
	}
}

func handleCharacterEnter(l logrus.FieldLogger, ctx context.Context, e _map2.StatusEvent[_map2.CharacterEnter]) {
	if e.Type != _map2.EventTopicMapStatusTypeCharacterEnter {
		return
	}

	err := instance.NewProcessor(l, ctx).HandleMapEnterAndEmit(e.Body.CharacterId, e.MapId, e.Instance, e.WorldId, e.ChannelId)
	if err != nil {
		l.WithError(err).Errorf("Error handling map enter for character [%d].", e.Body.CharacterId)
	}
}

func handleCharacterExit(l logrus.FieldLogger, ctx context.Context, e _map2.StatusEvent[_map2.CharacterExit]) {
	if e.Type != _map2.EventTopicMapStatusTypeCharacterExit {
		return
	}

	err := instance.NewProcessor(l, ctx).HandleMapExitAndEmit(e.Body.CharacterId, e.MapId, e.Instance, e.WorldId, e.ChannelId)
	if err != nil {
		l.WithError(err).Errorf("Error handling map exit for character [%d].", e.Body.CharacterId)
	}
}
