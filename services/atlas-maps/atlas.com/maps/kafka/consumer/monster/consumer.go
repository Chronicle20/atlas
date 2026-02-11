package monster

import (
	consumer2 "atlas-maps/kafka/consumer"
	monsterKafka "atlas-maps/kafka/message/monster"
	"atlas-maps/map/character"
	monster2 "atlas-maps/map/monster"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(monsterKafka.EnvEventTopicMonsterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(monsterKafka.EnvEventTopicMonsterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled)))
	}
}

func handleStatusEventKilled(l logrus.FieldLogger, ctx context.Context, event monsterKafka.StatusEvent[monsterKafka.StatusEventKilledBody]) {
	if event.Type != monsterKafka.EventMonsterStatusKilled {
		return
	}

	l.Debugf("Monster [%d] killed in world [%d] channel [%d] map [%d] instance [%s]. Resetting spawn cooldown.",
		event.MonsterId, event.WorldId, event.ChannelId, event.MapId, event.Instance)

	t := tenant.MustFromContext(ctx)
	f := field.NewBuilder(event.WorldId, event.ChannelId, event.MapId).SetInstance(event.Instance).Build()
	mapKey := character.MapKey{
		Tenant: t,
		Field:  f,
	}

	monster2.GetRegistry().ResetCooldown(mapKey, event.MonsterId)
}
