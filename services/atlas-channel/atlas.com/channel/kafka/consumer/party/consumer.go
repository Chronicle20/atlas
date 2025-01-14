package party

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/party"
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
)

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)("party_status_event")(EnvEventStatusTopic)(groupId)
	}
}

func CreatedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleCreatedEvent(sc, wp)))
	}
}

func handleCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[createdEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[createdEventBody]) {
		if e.Type != EventPartyStatusTypeCreated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		p, err := party.GetById(l)(ctx)(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("Received created event for party [%d] which does not exist.", e.PartyId)
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(p.LeaderId(), partyCreated(l)(ctx)(wp)(e.PartyId))
	}
}

func partyCreated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32) model.Operator[session.Model] {
			partyCreatedFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)
			return func(partyId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := partyCreatedFunc(s, writer.PartyCreatedBody(l)(partyId))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce party [%d] created to character [%d].", partyId, s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}

func LeftStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleLeftEvent(sc, wp)))
	}
}

func handleLeftEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[leftEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[leftEventBody]) {
		if e.Type != EventPartyStatusTypeLeft {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		p, err := party.GetById(l)(ctx)(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.GetById(l)(ctx)(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for character [%d] which does not exist.", e.Body.CharacterId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(m.Id(), partyLeft(l)(ctx)(wp)(p, tc, sc.ChannelId()))
			}
		}()
		go func() {
			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, partyLeft(l)(ctx)(wp)(p, tc, sc.ChannelId()))
		}()

	}
}

func partyLeft(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
			partyLeftFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)
			return func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := partyLeftFunc(s, writer.PartyLeftBody(l)(p, tc, forChannel))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] left party [%d].", s.CharacterId(), p.Id())
						return err
					}
					return nil
				}
			}
		}
	}
}

func DisbandStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleDisbandEvent(sc, wp)))
	}
}

func handleDisbandEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[disbandEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[disbandEventBody]) {
		if e.Type != EventPartyStatusTypeDisband {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		tc, err := character.GetById(l)(ctx)(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received disband event for character [%d] which does not exist.", e.Body.CharacterId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range e.Body.Members {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(m, partyDisband(l)(ctx)(wp)(e.PartyId, tc, sc.ChannelId()))
			}
		}()
		go func() {
			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, partyDisband(l)(ctx)(wp)(e.PartyId, tc, sc.ChannelId()))
		}()

	}
}

func partyDisband(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel byte) model.Operator[session.Model] {
			partyDisbandFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)
			return func(partyId uint32, tc character.Model, forChannel byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := partyDisbandFunc(s, writer.PartyDisbandBody(l)(partyId, tc, forChannel))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] left party [%d].", s.CharacterId(), partyId)
						return err
					}
					return nil
				}
			}
		}
	}
}

func JoinStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleJoinEvent(sc, wp)))
	}
}

func handleJoinEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[joinedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[joinedEventBody]) {
		if e.Type != EventPartyStatusTypeJoined {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		p, err := party.GetById(l)(ctx)(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.GetById(l)(ctx)(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received join event for character [%d] which does not exist.", e.Body.CharacterId)
			return
		}

		// For remaining party members.
		for _, m := range p.Members() {
			go func() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(m.Id(), partyJoined(l)(ctx)(wp)(p, tc, sc.ChannelId()))
			}()
		}
	}
}

func partyJoined(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
			partyJoinedFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)
			return func(p party.Model, tc character.Model, forChannel byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := partyJoinedFunc(s, writer.PartyJoinBody(l)(p, tc, forChannel))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] joined party [%d].", s.CharacterId(), p.Id())
						return err
					}
					return nil
				}
			}
		}
	}
}
