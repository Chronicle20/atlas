package _map

import (
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/pet"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvEventTopicMapStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(db))))
		}
	}
}

func handleStatusEventCharacterEnter(db *gorm.DB) message.Handler[statusEvent[characterEnter]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterEnter]) {
		if e.Type != EventTopicMapStatusTypeCharacterEnter {
			return
		}
		_ = pet.OwnerEnterMap(l)(ctx)(db)(e.Body.CharacterId, _map.NewModel(world.Id(e.WorldId))(channel.Id(e.ChannelId))(_map.Id(e.MapId)))
	}
}

func handleStatusEventCharacterExit(db *gorm.DB) message.Handler[statusEvent[characterExit]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterExit]) {
		if e.Type != EventTopicMapStatusTypeCharacterExit {
			return
		}
		_ = pet.OwnerExitMap(l)(ctx)(db)(e.Body.CharacterId, _map.NewModel(world.Id(e.WorldId))(channel.Id(e.ChannelId))(_map.Id(e.MapId)))
	}
}
