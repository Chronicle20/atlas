package invite

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	invite2 "atlas-channel/kafka/message/invite"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/invite"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	buddypkt "github.com/Chronicle20/atlas/libs/atlas-packet/buddy"
	guildpkt "github.com/Chronicle20/atlas/libs/atlas-packet/guild"
	guildcb "github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	messengerpkt "github.com/Chronicle20/atlas/libs/atlas-packet/messenger"
	messengercb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("invite_status_event")(invite2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(invite2.EnvEventStatusTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRejectedStatusEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleCreatedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[invite2.StatusEvent[invite2.CreatedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e invite2.StatusEvent[invite2.CreatedEventBody]) {
		if e.Type != invite.StatusTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		rc, err := character.NewProcessor(l, ctx).GetById()(uint32(e.Body.OriginatorId))
		if err != nil {
			l.WithError(err).Errorf("Unablet to get character [%d] details, who generated the invite.", e.Body.OriginatorId)
			return
		}

		var eventHandler model.Operator[session.Model]
		if e.InviteType == invite.TypeParty {
			eventHandler = handlePartyCreatedStatusEvent(l)(ctx)(wp)(uint32(e.ReferenceId), rc.Name())
		} else if e.InviteType == invite.TypeBuddy {
			eventHandler = handleBuddyCreatedStatusEvent(l)(ctx)(wp)(uint32(e.Body.TargetId), uint32(e.ReferenceId), rc.Name())
		} else if e.InviteType == invite.TypeGuild {
			eventHandler = handleGuildCreatedStatusEvent(l)(ctx)(wp)(uint32(e.ReferenceId), rc.Name())
		} else if e.InviteType == invite.TypeMessenger {
			eventHandler = handleMessengerCreatedStatusEvent(l)(ctx)(wp)(uint32(e.ReferenceId), rc.Name())
		}

		if eventHandler != nil {
			session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.Body.TargetId), eventHandler)
		}
	}
}

func handlePartyCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, originatorName string) model.Operator[session.Model] {
			return func(partyId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyInviteBody(partyId, originatorName))
			}
		}
	}
}

func handleBuddyCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
			return func(actorId uint32, originatorId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(buddypkt.BuddyOperationWriter)(buddypkt.BuddyInviteBody(actorId, originatorId, originatorName))
			}
		}
	}
}

func handleGuildCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
			return func(originatorId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(guildcb.GuildOperationWriter)(guildpkt.GuildInviteBody(originatorId, originatorName))
			}
		}
	}
}

func handleMessengerCreatedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(originatorId uint32, originatorName string) model.Operator[session.Model] {
			return func(originatorId uint32, originatorName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(messengercb.MessengerOperationWriter)(messengerpkt.MessengerOperationInviteBody(originatorName, originatorId))
			}
		}
	}
}

func handleRejectedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[invite2.StatusEvent[invite2.RejectedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e invite2.StatusEvent[invite2.RejectedEventBody]) {
		if e.Type != invite.StatusTypeRejected {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		rc, err := character.NewProcessor(l, ctx).GetById()(uint32(e.Body.TargetId))
		if err != nil {
			l.WithError(err).Errorf("Unablet to get character [%d] details, who generated the invite.", e.Body.OriginatorId)
			return
		}

		var eventHandler model.Operator[session.Model]
		if e.InviteType == invite.TypeParty {
			eventHandler = handlePartyRejectedStatusEvent(l)(ctx)(wp)(rc.Name())
		} else if e.InviteType == invite.TypeBuddy {
			// TODO send rejection to requesting character.
		} else if e.InviteType == invite.TypeGuild {
			eventHandler = handleGuildRejectedStatusEvent(l)(ctx)(wp)(rc.Name())
		} else if e.InviteType == invite.TypeMessenger {
			eventHandler = handleMessengerRejectedStatusEvent(l)(ctx)(wp)(rc.Name())
		}

		if eventHandler != nil {
			session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.Body.OriginatorId), eventHandler)
		}
	}
}

func handlePartyRejectedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
			return func(targetName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyErrorBody("HAVE_DENIED_REQUEST_TO_THE_PARTY", targetName))
			}
		}
	}
}

func handleGuildRejectedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
			return func(targetName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(guildcb.GuildOperationWriter)(guildpkt.GuildErrorBody2(guildpkt.GuildOperationInviteDenied, targetName))
			}
		}
	}
}

func handleMessengerRejectedStatusEvent(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(targetName string) model.Operator[session.Model] {
			return func(targetName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(messengercb.MessengerOperationWriter)(messengerpkt.MessengerOperationInviteDeclinedBody(targetName, 0))
			}
		}
	}
}
