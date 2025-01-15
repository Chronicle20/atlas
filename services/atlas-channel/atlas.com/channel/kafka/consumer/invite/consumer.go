package invite

import (
	"atlas-channel/character"
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
)

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)("invite_status_event")(EnvEventStatusTopic)(groupId)
	}
}

func CreatedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent(sc, wp)))
	}
}

func handleCreatedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[createdEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[createdEventBody]) {
		if e.Type != EventInviteStatusTypeCreated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		rc, err := character.GetById(l)(ctx)(e.Body.OriginatorId)
		if err != nil {
			l.WithError(err).Errorf("Unablet to get character [%d] details, who generated the invite.", e.Body.OriginatorId)
			return
		}

		var eventHandler model.Operator[session.Model]
		if e.InviteType == InviteTypeParty {
			eventHandler = handlePartyCreatedStatusEvent(l)(ctx)(wp)(e.ReferenceId, rc.Name())
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.TargetId, eventHandler)
	}
}

func handlePartyCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
			partyOperationFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)
			return func(partyId uint32, originatorName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := partyOperationFunc(s, writer.PartyInviteBody(l)(partyId, originatorName))
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}
}

func RejectedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleRejectedStatusEvent(sc, wp)))
	}
}

func handleRejectedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[rejectedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[rejectedEventBody]) {
		if e.Type != EventInviteStatusTypeRejected {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		rc, err := character.GetById(l)(ctx)(e.Body.TargetId)
		if err != nil {
			l.WithError(err).Errorf("Unablet to get character [%d] details, who generated the invite.", e.Body.OriginatorId)
			return
		}

		var eventHandler model.Operator[session.Model]
		if e.InviteType == InviteTypeParty {
			eventHandler = handlePartyRejectedStatusEvent(l)(ctx)(wp)(rc.Name())
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.OriginatorId, eventHandler)
	}
}

func handlePartyRejectedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
			partyOperationFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)
			return func(targetName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := partyOperationFunc(s, writer.PartyErrorBody(l)("HAVE_DENIED_REQUEST_TO_THE_PARTY", targetName))
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}
}
