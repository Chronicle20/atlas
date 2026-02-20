package member

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	member2 "atlas-channel/kafka/message/party/member"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
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
			rf(consumer2.NewConfig(l)("party_member_status_event")(member2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(member2.EnvEventStatusTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLoginEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLogoutEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleLoginEvent(sc server.Model, wp writer.Producer) message.Handler[member2.StatusEvent[member2.LoginEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e member2.StatusEvent[member2.LoginEventBody]) {
		if e.Type != member2.EventPartyStatusTypeLogin {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received event for character [%d] which does not exist.", e.CharacterId)
			return
		}

		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyUpdate(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] triggered party [%d] update.", m.Id(), p.Id())
				}
			}
		}()
	}
}

func handleLogoutEvent(sc server.Model, wp writer.Producer) message.Handler[member2.StatusEvent[member2.LogoutEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e member2.StatusEvent[member2.LogoutEventBody]) {
		if e.Type != member2.EventPartyStatusTypeLogout {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received event for character [%d] which does not exist.", e.CharacterId)
			return
		}

		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyUpdate(l)(ctx)(wp)(p, tc, sc.ChannelId()))
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
