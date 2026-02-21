package _map

import (
	consumer2 "atlas-monsters/kafka/consumer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicMapStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventCharacterEnter(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterEnter]) {
	if e.Type != EventTopicMapStatusTypeCharacterEnter {
		return
	}

	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

	p := monster.NewProcessor(l, ctx)
	provider := p.NotControlledInFieldProvider(f)
	_ = model.ForEachSlice(provider, p.FindNextController(_map.CharacterIdsInFieldProvider(l)(ctx)(f)), model.ParallelExecute())
}

func handleStatusEventCharacterExit(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterExit]) {
	if e.Type != EventTopicMapStatusTypeCharacterExit {
		return
	}

	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

	ocids, err := _map.CharacterIdsInFieldProvider(l)(ctx)(f)()
	if err != nil {
		return
	}

	p := monster.NewProcessor(l, ctx)
	provider := p.ControlledByCharacterInFieldProvider(f, e.Body.CharacterId)
	_ = model.ForEachSlice(provider, p.StopControl, model.ParallelExecute())
	_ = model.ForEachSlice(provider, p.FindNextController(model.FixedProvider(ocids)), model.ParallelExecute())
}
