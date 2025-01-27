package thread

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
	"atlas-guilds/thread"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("guild_thread_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandCreate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandUpdate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandDelete(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandAddReply(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandDeleteReply(db))))

		}
	}
}

func handleCommandCreate(db *gorm.DB) message.Handler[command[createCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
		if c.Type != CommandTypeCreate {
			return
		}
		g, err := guild.GetById(l)(ctx)(db)(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.Create(l)(ctx)(db)(g.WorldId(), g.Id(), c.CharacterId, c.Body.Title, c.Body.Message, c.Body.EmoticonId, c.Body.Notice)
	}
}

func handleCommandUpdate(db *gorm.DB) message.Handler[command[updateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[updateCommandBody]) {
		if c.Type != CommandTypeUpdate {
			return
		}
		g, err := guild.GetById(l)(ctx)(db)(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.Update(l)(ctx)(db)(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId, c.Body.Title, c.Body.Message, c.Body.EmoticonId, c.Body.Notice)
	}
}

func handleCommandDelete(db *gorm.DB) message.Handler[command[deleteCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[deleteCommandBody]) {
		if c.Type != CommandTypeDelete {
			return
		}
		g, err := guild.GetById(l)(ctx)(db)(c.GuildId)
		if err != nil {
			return
		}
		_ = thread.Delete(l)(ctx)(db)(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId)
	}
}

func handleCommandAddReply(db *gorm.DB) message.Handler[command[addReplyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[addReplyCommandBody]) {
		if c.Type != CommandTypeAddReply {
			return
		}
		g, err := guild.GetById(l)(ctx)(db)(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.Reply(l)(ctx)(db)(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId, c.Body.Message)
	}
}

func handleCommandDeleteReply(db *gorm.DB) message.Handler[command[deleteReplyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[deleteReplyCommandBody]) {
		if c.Type != CommandTypeDeleteReply {
			return
		}
		g, err := guild.GetById(l)(ctx)(db)(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.DeleteReply(l)(ctx)(db)(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId, c.Body.ReplyId)
	}
}
