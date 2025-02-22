package invite

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
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
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("invite_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventStatusTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRejectedStatusEvent(sc, wp))))
			}
		}
	}
}

func handleCreatedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[createdEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[createdEventBody]) {
		if e.Type != EventInviteStatusTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		rc, err := character.GetById(l)(ctx)()(e.Body.OriginatorId)
		if err != nil {
			l.WithError(err).Errorf("Unablet to get character [%d] details, who generated the invite.", e.Body.OriginatorId)
			return
		}

		var eventHandler model.Operator[session.Model]
		if e.InviteType == InviteTypeParty {
			eventHandler = handlePartyCreatedStatusEvent(l)(ctx)(wp)(e.ReferenceId, rc.Name())
		} else if e.InviteType == InviteTypeBuddy {
			eventHandler = handleBuddyCreatedStatusEvent(l)(ctx)(wp)(e.Body.TargetId, e.ReferenceId, rc.Name())
		} else if e.InviteType == InviteTypeGuild {
			eventHandler = handleGuildCreatedStatusEvent(l)(ctx)(wp)(e.ReferenceId, rc.Name())
		}

		if eventHandler != nil {
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.TargetId, eventHandler)
		}
	}
}

func handlePartyCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
			return func(partyId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyInviteBody(l)(partyId, originatorName))
			}
		}
	}
}

func handleBuddyCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
			return func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.BuddyOperation)(writer.BuddyInviteBody(l, t)(actorId, originatorId, originatorName))
			}
		}
	}
}

func handleGuildCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
			return func(originatorId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildInviteBody(l)(originatorId, originatorName))
			}
		}
	}
}

func handleRejectedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[statusEvent[rejectedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[rejectedEventBody]) {
		if e.Type != EventInviteStatusTypeRejected {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		rc, err := character.GetById(l)(ctx)()(e.Body.TargetId)
		if err != nil {
			l.WithError(err).Errorf("Unablet to get character [%d] details, who generated the invite.", e.Body.OriginatorId)
			return
		}

		var eventHandler model.Operator[session.Model]
		if e.InviteType == InviteTypeParty {
			eventHandler = handlePartyRejectedStatusEvent(l)(ctx)(wp)(rc.Name())
		} else if e.InviteType == InviteTypeBuddy {
			// TODO send rejection to requesting character.
		} else if e.InviteType == InviteTypeGuild {
			eventHandler = handleGuildRejectedStatusEvent(l)(ctx)(wp)(rc.Name())
		}

		if eventHandler != nil {
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.OriginatorId, eventHandler)
		}
	}
}

func handlePartyRejectedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
			return func(targetName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.PartyOperation)(writer.PartyErrorBody(l)("HAVE_DENIED_REQUEST_TO_THE_PARTY", targetName))
			}
		}
	}
}

func handleGuildRejectedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
			return func(targetName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildErrorBody2(l)(writer.GuildOperationInviteDenied, targetName))
			}
		}
	}
}
