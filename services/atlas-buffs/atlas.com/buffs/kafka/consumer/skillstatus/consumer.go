package skillstatus

import (
	"atlas-buffs/berserk"
	consumer2 "atlas-buffs/kafka/consumer"
	skillstatus2 "atlas-buffs/kafka/message/skillstatus"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("skill_status_event")(skillstatus2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(skillstatus2.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventUpdated))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventUpdated(l logrus.FieldLogger, ctx context.Context, e skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]) {
	if e.Type != skillstatus2.StatusEventTypeUpdated {
		return
	}
	if e.SkillId != uint32(skill.DarkKnightBerserkId) {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleSkillUpdated(e.WorldId, e.CharacterId, e.Body.Level); err != nil {
		l.WithError(err).Errorf("Unable to process berserk skill update for character [%d].", e.CharacterId)
	}
}

func handleStatusEventDeleted(l logrus.FieldLogger, ctx context.Context, e skillstatus2.StatusEvent[skillstatus2.StatusEventDeletedBody]) {
	if e.Type != skillstatus2.StatusEventTypeDeleted {
		return
	}
	if e.SkillId != uint32(skill.DarkKnightBerserkId) {
		return
	}
	if err := berserk.NewProcessor(l, ctx).Untrack(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to untrack berserk for character [%d] after skill deletion.", e.CharacterId)
	}
}
