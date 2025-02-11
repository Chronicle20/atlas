package skill

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"time"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("skill_status_event")(EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdated(sc, wp))))
			}
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
		if e.Type != StatusEventTypeCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, announceSkillUpdate(l)(ctx)(wp)(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
	}
}

func handleUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventUpdatedBody]) {
		if e.Type != StatusEventTypeUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, announceSkillUpdate(l)(ctx)(wp)(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
	}
}

func announceSkillUpdate(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
			return func(skillId uint32, level byte, masterLevel byte, expiration time.Time) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.CharacterSkillChange)(s, writer.CharacterSkillChangeBody(l, tenant.MustFromContext(ctx))(true, skillId, level, masterLevel, expiration, true))
					if err != nil {
						l.WithError(err).Errorf("Unable to update character [%d] skill [%d].", s.CharacterId(), skillId)
					}
					return err
				}
			}
		}
	}
}
