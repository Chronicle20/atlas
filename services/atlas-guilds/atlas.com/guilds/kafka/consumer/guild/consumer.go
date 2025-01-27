package guild

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("guild_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCreate(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandCreationAgreement(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandChangeEmblem(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandChangeNotice(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandLeave(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestInvite(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandChangeTitles(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandChangeMemberTitle(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestDisband(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCapacityIncrease(db))))
		}
	}
}

func handleCommandRequestCreate(db *gorm.DB) message.Handler[command[requestCreateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestCreateBody]) {
		if c.Type != CommandTypeRequestCreate {
			return
		}

		_ = guild.RequestCreate(l)(ctx)(db)(c.CharacterId, c.Body.WorldId, c.Body.ChannelId, c.Body.MapId, c.Body.Name)
	}
}

func handleCommandCreationAgreement(db *gorm.DB) message.Handler[command[creationAgreementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[creationAgreementBody]) {
		if c.Type != CommandTypeCreationAgreement {
			return
		}

		_ = guild.CreationAgreementResponse(l)(ctx)(db)(c.CharacterId, c.Body.Agreed)
	}
}

func handleCommandChangeEmblem(db *gorm.DB) message.Handler[command[changeEmblemBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeEmblemBody]) {
		if c.Type != CommandTypeChangeEmblem {
			return
		}

		_ = guild.ChangeEmblem(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.Logo, c.Body.LogoColor, c.Body.LogoBackground, c.Body.LogoBackgroundColor)
	}
}

func handleCommandChangeNotice(db *gorm.DB) message.Handler[command[changeNoticeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeNoticeBody]) {
		if c.Type != CommandTypeChangeNotice {
			return
		}

		_ = guild.ChangeNotice(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.Notice)
	}
}

func handleCommandLeave(db *gorm.DB) message.Handler[command[leaveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[leaveBody]) {
		if c.Type != CommandTypeLeave {
			return
		}

		_ = guild.Leave(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.Force)
	}
}

func handleCommandRequestInvite(db *gorm.DB) message.Handler[command[requestInviteBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestInviteBody]) {
		if c.Type != CommandTypeRequestInvite {
			return
		}

		_ = guild.RequestInvite(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.TargetId)
	}
}

func handleCommandChangeTitles(db *gorm.DB) message.Handler[command[changeTitlesBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeTitlesBody]) {
		if c.Type != CommandTypeChangeTitles {
			return
		}

		_ = guild.ChangeTitles(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.Titles)
	}
}

func handleCommandChangeMemberTitle(db *gorm.DB) message.Handler[command[changeMemberTitleBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeMemberTitleBody]) {
		if c.Type != CommandTypeChangeMemberTitle {
			return
		}

		_ = guild.ChangeMemberTitle(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.TargetId, c.Body.Title)
	}
}

func handleCommandRequestDisband(db *gorm.DB) message.Handler[command[requestDisbandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestDisbandBody]) {
		if c.Type != CommandTypeRequestDisband {
			return
		}

		_ = guild.RequestDisband(l)(ctx)(db)(c.CharacterId)
	}
}

func handleCommandRequestCapacityIncrease(db *gorm.DB) message.Handler[command[requestCapacityIncreaseBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestCapacityIncreaseBody]) {
		if c.Type != CommandTypeRequestCapacityIncrease {
			return
		}

		_ = guild.RequestCapacityIncrease(l)(ctx)(db)(c.CharacterId)
	}
}
