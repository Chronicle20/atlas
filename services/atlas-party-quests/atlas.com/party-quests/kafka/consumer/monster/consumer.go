package monster

import (
	consumer2 "atlas-party-quests/kafka/consumer"
	"atlas-party-quests/instance"
	monsterMessage "atlas-party-quests/kafka/message/monster"
	"context"

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
			rf(consumer2.NewConfig(l)("monster_status_event")(monsterMessage.EnvEventTopicMonsterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(monsterMessage.EnvEventTopicMonsterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDamaged(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFriendlyDrop(db)))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventDamaged(db *gorm.DB) message.Handler[monsterMessage.StatusEvent[monsterMessage.DamagedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterMessage.StatusEvent[monsterMessage.DamagedBody]) {
		if e.Type != monsterMessage.EventStatusDamaged {
			return
		}

		err := instance.NewProcessor(l, ctx, db).HandleFriendlyMonsterDamagedAndEmit(e.Field(), e.MonsterId)
		if err != nil {
			l.Debugf("Monster [%d] damaged in field [%s] not relevant to any PQ: %s.", e.MonsterId, e.Field().Id(), err.Error())
		}
	}
}

func handleStatusEventKilled(db *gorm.DB) message.Handler[monsterMessage.StatusEvent[monsterMessage.KilledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterMessage.StatusEvent[monsterMessage.KilledBody]) {
		if e.Type != monsterMessage.EventStatusKilled {
			return
		}

		err := instance.NewProcessor(l, ctx, db).HandleFriendlyMonsterKilledAndEmit(e.Field(), e.MonsterId)
		if err != nil {
			l.Debugf("Monster [%d] killed in field [%s] not relevant to any PQ: %s.", e.MonsterId, e.Field().Id(), err.Error())
		}
	}
}

func handleStatusEventFriendlyDrop(db *gorm.DB) message.Handler[monsterMessage.StatusEvent[monsterMessage.FriendlyDropBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterMessage.StatusEvent[monsterMessage.FriendlyDropBody]) {
		if e.Type != monsterMessage.EventStatusFriendlyDrop {
			return
		}

		err := instance.NewProcessor(l, ctx, db).HandleFriendlyMonsterDropAndEmit(e.Field(), e.MonsterId, e.Body.ItemCount)
		if err != nil {
			l.Debugf("Friendly drop for monster [%d] in field [%s] not relevant to any PQ: %s.", e.MonsterId, e.Field().Id(), err.Error())
		}
	}
}
