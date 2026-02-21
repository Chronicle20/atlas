package buff

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	consumer2 "atlas-channel/kafka/consumer"
	buff2 "atlas-channel/kafka/message/buff"
	_map "atlas-channel/map"
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
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(buff2.EnvEventStatusTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleStatusEventApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(writer.CharacterBuffGive)(writer.CharacterBuffGiveBody(l)(ctx)(bs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
				err = session.Announce(l)(ctx)(wp)(writer.CharacterBuffGiveForeign)(writer.CharacterBuffGiveForeignBody(l)(ctx)(e.CharacterId, bs))(os)
				if err != nil {
					l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
					return err
				}
				return nil
			})
			return nil
		})
	}
}

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			ebs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			ebs = append(ebs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(writer.CharacterBuffCancel)(writer.CharacterBuffCancelBody(l)(ctx)(ebs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
				err = session.Announce(l)(ctx)(wp)(writer.CharacterBuffCancelForeign)(writer.CharacterBuffCancelForeignBody(l)(ctx)(e.CharacterId, ebs))(os)
				if err != nil {
					l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
					return err
				}
				return nil
			})
			return nil
		})
	}
}
