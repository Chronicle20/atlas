package thread

import (
	"atlas-channel/guild/thread"
	consumer2 "atlas-channel/kafka/consumer"
	thread2 "atlas-channel/kafka/message/guild/thread"
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
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("guild_thread_status_event")(thread2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(thread2.EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadCreated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadReplyAdded(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleThreadReplyDeleted(sc, wp))))
			}
		}
	}
}

func handleThreadCreated(sc server.Model, wp writer.Producer) message.Handler[thread2.StatusEvent[thread2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e thread2.StatusEvent[thread2.CreatedStatusEventBody]) {
		if e.Type != thread2.StatusEventTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
		if err != nil {
			l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", e.ThreadId, e.ActorId)
		}
	}
}

func handleThreadUpdated(sc server.Model, wp writer.Producer) message.Handler[thread2.StatusEvent[thread2.UpdatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e thread2.StatusEvent[thread2.UpdatedStatusEventBody]) {
		if e.Type != thread2.StatusEventTypeUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
		if err != nil {
			l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", e.ThreadId, e.ActorId)
		}
	}
}

func handleThreadReplyAdded(sc server.Model, wp writer.Producer) message.Handler[thread2.StatusEvent[thread2.ReplyAddedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e thread2.StatusEvent[thread2.ReplyAddedStatusEventBody]) {
		if e.Type != thread2.StatusEventTypeReplyAdded {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
		if err != nil {
			l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", e.ThreadId, e.ActorId)
		}
	}
}

func handleThreadReplyDeleted(sc server.Model, wp writer.Producer) message.Handler[thread2.StatusEvent[thread2.ReplyDeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e thread2.StatusEvent[thread2.ReplyDeletedStatusEventBody]) {
		if e.Type != thread2.StatusEventTypeReplyDeleted {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.ActorId, refreshThread(l)(ctx)(wp)(e.GuildId, e.ThreadId))
		if err != nil {
			l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", e.ThreadId, e.ActorId)
		}
	}
}

func refreshThread(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, threadId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, threadId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, threadId uint32) model.Operator[session.Model] {
			return func(guildId uint32, threadId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					t, err := thread.NewProcessor(l, ctx).GetById(guildId, threadId)
					if err != nil {
						return err
					}
					return session.Announce(l)(ctx)(wp)(writer.GuildBBS)(writer.GuildBBSThreadBody(l)(t))(s)
				}
			}
		}
	}
}
