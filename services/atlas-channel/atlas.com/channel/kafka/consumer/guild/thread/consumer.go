package thread

import (
	"atlas-channel/guild/thread"
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
			rf(consumer2.NewConfig(l)("guild_thread_status_event")(EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadCreated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadReplyAdded(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadReplyDeleted(sc, wp))))
			}
		}
	}
}

func handleThreadCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[createdStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[createdStatusEventBody]) {
		if e.Type != StatusEventTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
	}
}

func handleThreadUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[updatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[updatedStatusEventBody]) {
		if e.Type != StatusEventTypeUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
	}
}

func handleThreadReplyAdded(sc server.Model, wp writer.Producer) message.Handler[statusEvent[replyAddedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[replyAddedStatusEventBody]) {
		if e.Type != StatusEventTypeReplyAdded {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
	}
}

func handleThreadReplyDeleted(sc server.Model, wp writer.Producer) message.Handler[statusEvent[replyDeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[replyDeletedStatusEventBody]) {
		if e.Type != StatusEventTypeReplyDeleted {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
	}
}

func refreshThread(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, threadId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, threadId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, threadId uint32) model.Operator[session.Model] {
			guildBBSFunc := session.Announce(l)(ctx)(wp)(writer.GuildBBS)
			return func(guildId uint32, threadId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					t, err := thread.GetById(l)(ctx)(guildId, threadId)
					if err != nil {
						l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", t.Id(), s.CharacterId())
						return err
					}
					err = guildBBSFunc(s, writer.GuildBBSThreadBody(l)(t))
					if err != nil {
						l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", t.Id(), s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}
