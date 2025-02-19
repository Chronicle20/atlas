package buff

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
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
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
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

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(writer.CharacterBuffGive)(s, writer.CharacterBuffGiveBody(l)(ctx)(bs))
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] buffs.", e.CharacterId)
			}
			return nil
		})
	}
}

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[statusEvent[expiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[expiredStatusEventBody]) {
		if e.Type != EventStatusTypeBuffExpired {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			ebs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			ebs = append(ebs, buff.NewBuff(e.Body.SourceId, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(writer.CharacterBuffCancel)(s, writer.CharacterBuffCancelBody(l)(ctx)(ebs))
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}
			return nil
		})
	}
}
