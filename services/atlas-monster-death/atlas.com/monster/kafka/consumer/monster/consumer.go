package monster

import (
	consumer2 "atlas-monster-death/kafka/consumer"
	"atlas-monster-death/monster"
	"context"
	"sync"

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
			rf(consumer2.NewConfig(l)("monster_status_event")(EnvEventTopicMonsterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicMonsterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleKilledStatusEvent)))
	}
}

func handleKilledStatusEvent(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventKilledBody]) {
	if e.Type != EventMonsterStatusKilled {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := monster.CreateDrops(l)(ctx)(e.WorldId, e.ChannelId, e.MapId, e.UniqueId, e.MonsterId, e.Body.X, e.Body.Y, e.Body.ActorId)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"worldId":   e.WorldId,
				"channelId": e.ChannelId,
				"mapId":     e.MapId,
				"monsterId": e.MonsterId,
			}).Error("Failed to create drops for monster death.")
		}
	}()

	go func() {
		defer wg.Done()
		dms, err := model.SliceMap(func(m damageEntry) (monster.DamageEntryModel, error) {
			return monster.NewDamageEntryModel(m.CharacterId, m.Damage), nil
		})(model.FixedProvider(e.Body.DamageEntries))(model.ParallelMap())()
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"worldId":   e.WorldId,
				"channelId": e.ChannelId,
				"mapId":     e.MapId,
				"monsterId": e.MonsterId,
			}).Error("Failed to map damage entries.")
			return
		}

		err = monster.DistributeExperience(l)(ctx)(e.WorldId, e.ChannelId, e.MapId, e.MonsterId, dms)
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"worldId":   e.WorldId,
				"channelId": e.ChannelId,
				"mapId":     e.MapId,
				"monsterId": e.MonsterId,
			}).Error("Failed to distribute experience for monster death.")
		}
	}()

	wg.Wait()
}
