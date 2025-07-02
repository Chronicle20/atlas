package thread

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
	thread2 "atlas-guilds/kafka/message/thread"
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
			rf(consumer2.NewConfig(l)("guild_thread_command")(thread2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(thread2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandCreate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandUpdate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandDelete(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandAddReply(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandDeleteReply(db))))

		}
	}
}

func handleCommandCreate(db *gorm.DB) message.Handler[thread2.Command[thread2.CreateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c thread2.Command[thread2.CreateCommandBody]) {
		if c.Type != thread2.CommandTypeCreate {
			return
		}
		g, err := guild.NewProcessor(l, ctx, db).GetById(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.NewProcessor(l, ctx, db).Create(g.WorldId(), g.Id(), c.CharacterId, c.Body.Title, c.Body.Message, c.Body.EmoticonId, c.Body.Notice)
	}
}

func handleCommandUpdate(db *gorm.DB) message.Handler[thread2.Command[thread2.UpdateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c thread2.Command[thread2.UpdateCommandBody]) {
		if c.Type != thread2.CommandTypeUpdate {
			return
		}
		g, err := guild.NewProcessor(l, ctx, db).GetById(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.NewProcessor(l, ctx, db).Update(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId, c.Body.Title, c.Body.Message, c.Body.EmoticonId, c.Body.Notice)
	}
}

func handleCommandDelete(db *gorm.DB) message.Handler[thread2.Command[thread2.DeleteCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c thread2.Command[thread2.DeleteCommandBody]) {
		if c.Type != thread2.CommandTypeDelete {
			return
		}
		g, err := guild.NewProcessor(l, ctx, db).GetById(c.GuildId)
		if err != nil {
			return
		}
		_ = thread.NewProcessor(l, ctx, db).Delete(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId)
	}
}

func handleCommandAddReply(db *gorm.DB) message.Handler[thread2.Command[thread2.AddReplyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c thread2.Command[thread2.AddReplyCommandBody]) {
		if c.Type != thread2.CommandTypeAddReply {
			return
		}
		g, err := guild.NewProcessor(l, ctx, db).GetById(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.NewProcessor(l, ctx, db).Reply(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId, c.Body.Message)
	}
}

func handleCommandDeleteReply(db *gorm.DB) message.Handler[thread2.Command[thread2.DeleteReplyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c thread2.Command[thread2.DeleteReplyCommandBody]) {
		if c.Type != thread2.CommandTypeDeleteReply {
			return
		}
		g, err := guild.NewProcessor(l, ctx, db).GetById(c.GuildId)
		if err != nil {
			return
		}
		_, _ = thread.NewProcessor(l, ctx, db).DeleteReply(g.WorldId(), g.Id(), c.Body.ThreadId, c.CharacterId, c.Body.ReplyId)
	}
}
