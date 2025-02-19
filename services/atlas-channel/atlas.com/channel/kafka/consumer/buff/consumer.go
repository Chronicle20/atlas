package buff

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/server"
	"atlas-channel/session"
	skill2 "atlas-channel/skill"
	"atlas-channel/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"errors"
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
			bs, err := buff.GetByCharacterId(l)(ctx)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve active buffs for character [%d].", s.CharacterId())
			}
			err = session.Announce(l)(ctx)(wp)(writer.CharacterBuffGive)(s, writer.CharacterBuffGiveBody(l)(ctx)(bs))
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
			ss, err := skill.GetByCharacterId(l)(ctx)(s.CharacterId())
			if err != nil {
				return err
			}
			var sm skill.Model
			for _, rs := range ss {
				if rs.Id() == e.Body.SourceId {
					sm = rs
				}
			}
			if sm.Id() != e.Body.SourceId {
				return errors.New("does not possess skill")
			}
			se, err := skill2.GetEffect(l)(ctx)(sm.Id(), sm.Level())
			if err != nil {
				return err
			}

			ebs := getExpiredBuffEffects(sm.Id(), se)

			bs, err := buff.GetByCharacterId(l)(ctx)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve active buffs for character [%d].", s.CharacterId())
				return err
			}

			res := append(bs, ebs)

			err = session.Announce(l)(ctx)(wp)(writer.CharacterBuffCancel)(s, writer.CharacterBuffCancelBody(l)(ctx)(res))
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}
			return nil
		})
	}
}

func getExpiredBuffEffects(skillId uint32, se effect.Model) buff.Model {
	changes := make([]stat.Model, 0)
	for _, su := range se.StatUps() {
		changes = append(changes, stat.NewStat(su.Mask(), su.Amount()))
	}

	return buff.NewBuff(skillId, se.Duration(), changes)
}
