package party

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	party2 "atlas-channel/kafka/message/party"
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
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_status_event")(party2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(party2.EnvEventStatusTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeft(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleExpel(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDisband(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleJoin(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeLeader(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
			}
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.CreatedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.CreatedEventBody]) {
		if e.Type != party2.EventPartyStatusTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("Received created event for party [%d] which does not exist.", e.PartyId)
			return
		}

		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(p.LeaderId(), partyCreated(l)(ctx)(wp)(e.PartyId))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce party [%d] created to character [%d].", e.PartyId, p.LeaderId())
		}
	}
}

func partyCreated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32) model.Operator[session.Model] {
			return func(partyId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyCreatedBody(l)(partyId))
			}
		}
	}
}

func handleLeft(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.LeftEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.LeftEventBody]) {
		if e.Type != party2.EventPartyStatusTypeLeft {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyLeft(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has left party [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, partyLeft(l)(ctx)(wp)(p, tc, sc.ChannelId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] has left party [%d].", tc.Id(), p.Id())
			}
		}()

	}
}

func partyLeft(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyLeftBody(l)(p, tc, forChannel))
			}
		}
	}
}

func handleExpel(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.ExpelEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.ExpelEventBody]) {
		if e.Type != party2.EventPartyStatusTypeExpel {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received expel event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received expel event for character [%d] which does not exist.", e.Body.CharacterId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyExpel(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] was expelled from party [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, partyExpel(l)(ctx)(wp)(p, tc, sc.ChannelId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] was expelled from party [%d].", tc.Id(), p.Id())
			}
		}()

	}
}

func partyExpel(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyExpelBody(l)(p, tc, forChannel))
			}
		}
	}
}

func handleDisband(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.DisbandEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.DisbandEventBody]) {
		if e.Type != party2.EventPartyStatusTypeDisband {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received disband event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range e.Body.Members {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m, partyDisband(l)(ctx)(wp)(e.PartyId, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] the party [%d] was disbanded.", m, e.PartyId)
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, partyDisband(l)(ctx)(wp)(e.PartyId, tc, sc.ChannelId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] the party [%d] was disbanded.", e.ActorId, e.PartyId)
			}
		}()

	}
}

func partyDisband(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyDisbandBody(l)(partyId, tc, forChannel))
			}
		}
	}
}

func handleJoin(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.JoinedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.JoinedEventBody]) {
		if e.Type != party2.EventPartyStatusTypeJoined {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received join event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining party members.
		for _, m := range p.Members() {
			go func() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyJoined(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] joined party [%d].", e.PartyId, p.Id())
				}
			}()
		}
	}
}

func partyJoined(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyJoinBody(l)(p, tc, forChannel))
			}
		}
	}
}

func handleChangeLeader(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.ChangeLeaderEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.ChangeLeaderEventBody]) {
		if e.Type != party2.EventPartyStatusTypeChangeLeader {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received expel event for party [%d] which does not exist.", e.PartyId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyChangeLeader(l)(ctx)(wp)(e.PartyId, e.Body.CharacterId, e.Body.Disconnected))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce change party [%d] leadership to [%d].", e.PartyId, e.Body.CharacterId)
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, partyChangeLeader(l)(ctx)(wp)(e.PartyId, e.Body.CharacterId, e.Body.Disconnected))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce change party [%d] leadership to [%d].", e.PartyId, e.Body.CharacterId)
			}
		}()

	}
}

func partyChangeLeader(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
			return func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyChangeLeaderBody(l)(targetCharacterId, disconnected))
			}
		}
	}
}

func handleError(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.ErrorEventBody]) {
		if e.Type != party2.EventPartyStatusTypeError {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, partyError(l)(ctx)(wp)(e.Body.Type, e.Body.CharacterName))
	}
}

func partyError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
			return func(errorType string, name string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyErrorBody(l)(errorType, name))
			}
		}
	}
}
