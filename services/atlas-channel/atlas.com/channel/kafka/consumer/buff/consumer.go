package buff

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("character_buff_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventStatusTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired(sc, wp))))
			}
		}
	}
}

func handleStatusEventApplied(sc server.Model, wp writer.Producer) message.Handler[statusEvent[appliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[appliedStatusEventBody]) {
		if e.Type != EventStatusTypeBuffApplied {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(writer.CharacterBuffGive)(writer.CharacterBuffGiveBody(l)(ctx)(bs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
			}

			_ = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), func(os session.Model) error {
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

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[statusEvent[expiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[expiredStatusEventBody]) {
		if e.Type != EventStatusTypeBuffExpired {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			ebs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			ebs = append(ebs, buff.NewBuff(e.Body.SourceId, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(writer.CharacterBuffCancel)(writer.CharacterBuffCancelBody(l)(ctx)(ebs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}

			_ = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), func(os session.Model) error {
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
