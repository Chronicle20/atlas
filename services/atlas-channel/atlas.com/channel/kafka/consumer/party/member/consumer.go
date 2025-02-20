package member

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("party_member_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventStatusTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLoginEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLogoutEvent(sc, wp))))
			}
		}
	}
}

func handleLoginEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[loginEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[loginEventBody]) {
		if e.Type != EventPartyStatusTypeLogin {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.GetById(l)(ctx)(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received event for character [%d] which does not exist.", e.CharacterId)
			return
		}

		go func() {
			for _, m := range p.Members() {
				err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(m.Id(), partyUpdate(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] triggered party [%d] update.", m.Id(), p.Id())
				}
			}
		}()
	}
}

func handleLogoutEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[logoutEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[logoutEventBody]) {
		if e.Type != EventPartyStatusTypeLogout {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.GetById(l)(ctx)(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received event for character [%d] which does not exist.", e.CharacterId)
			return
		}

		go func() {
			for _, m := range p.Members() {
				err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(m.Id(), partyUpdate(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] triggered party [%d] update.", m.Id(), p.Id())
				}
			}
		}()
	}
}

func partyUpdate(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyUpdateBody(l)(p, tc, forChannel))
			}
		}
	}
}
