package guild

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const consumerCommand = "guild_command"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func RequestCreateRegister(db *gorm.DB) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, c command[requestCreateBody]) {
			if c.Type != CommandTypeRequestCreate {
				return
			}

			_ = guild.RequestCreate(l)(ctx)(db)(c.CharacterId, c.Body.WorldId, c.Body.ChannelId, c.Body.MapId, c.Body.Name)
		}))
	}
}

func CreationAgreementRegister(db *gorm.DB) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, c command[creationAgreementBody]) {
			if c.Type != CommandTypeCreationAgreement {
				return
			}

			_ = guild.CreationAgreementResponse(l)(ctx)(db)(c.CharacterId, c.Body.Agreed)
		}))
	}
}

func ChangeEmblemRegister(db *gorm.DB) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, c command[changeEmblemBody]) {
			if c.Type != CommandTypeChangeEmblem {
				return
			}

			_ = guild.ChangeEmblem(l)(ctx)(db)(c.Body.GuildId, c.CharacterId, c.Body.Logo, c.Body.LogoColor, c.Body.LogoBackground, c.Body.LogoBackgroundColor)
		}))
	}
}
