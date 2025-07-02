package guild

import (
	"atlas-guilds/guild"
	consumer2 "atlas-guilds/kafka/consumer"
	guild2 "atlas-guilds/kafka/message/guild"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("guild_command")(guild2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(guild2.EnvCommandTopic)()
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

func handleCommandRequestCreate(db *gorm.DB) message.Handler[guild2.Command[guild2.RequestCreateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.RequestCreateBody]) {
		if c.Type != guild2.CommandTypeRequestCreate {
			return
		}

		f := field.NewBuilder(world.Id(c.Body.WorldId), channel.Id(c.Body.ChannelId), _map.Id(c.Body.MapId)).Build()
		_ = guild.NewProcessor(l, ctx, db).RequestCreateAndEmit(c.CharacterId, f, c.Body.Name, c.TransactionId)
	}
}

func handleCommandCreationAgreement(db *gorm.DB) message.Handler[guild2.Command[guild2.CreationAgreementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.CreationAgreementBody]) {
		if c.Type != guild2.CommandTypeCreationAgreement {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).CreationAgreementResponseAndEmit(c.CharacterId, c.Body.Agreed, c.TransactionId)
	}
}

func handleCommandChangeEmblem(db *gorm.DB) message.Handler[guild2.Command[guild2.ChangeEmblemBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.ChangeEmblemBody]) {
		if c.Type != guild2.CommandTypeChangeEmblem {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).ChangeEmblemAndEmit(c.Body.GuildId, c.CharacterId, c.Body.Logo, c.Body.LogoColor, c.Body.LogoBackground, c.Body.LogoBackgroundColor, c.TransactionId)
	}
}

func handleCommandChangeNotice(db *gorm.DB) message.Handler[guild2.Command[guild2.ChangeNoticeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.ChangeNoticeBody]) {
		if c.Type != guild2.CommandTypeChangeNotice {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).ChangeNoticeAndEmit(c.Body.GuildId, c.CharacterId, c.Body.Notice, c.TransactionId)
	}
}

func handleCommandLeave(db *gorm.DB) message.Handler[guild2.Command[guild2.LeaveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.LeaveBody]) {
		if c.Type != guild2.CommandTypeLeave {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).LeaveAndEmit(c.Body.GuildId, c.CharacterId, c.Body.Force, c.TransactionId)
	}
}

func handleCommandRequestInvite(db *gorm.DB) message.Handler[guild2.Command[guild2.RequestInviteBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.RequestInviteBody]) {
		if c.Type != guild2.CommandTypeRequestInvite {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).RequestInviteAndEmit(c.Body.GuildId, c.CharacterId, c.Body.TargetId)
	}
}

func handleCommandChangeTitles(db *gorm.DB) message.Handler[guild2.Command[guild2.ChangeTitlesBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.ChangeTitlesBody]) {
		if c.Type != guild2.CommandTypeChangeTitles {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).ChangeTitlesAndEmit(c.Body.GuildId, c.CharacterId, c.Body.Titles, c.TransactionId)
	}
}

func handleCommandChangeMemberTitle(db *gorm.DB) message.Handler[guild2.Command[guild2.ChangeMemberTitleBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.ChangeMemberTitleBody]) {
		if c.Type != guild2.CommandTypeChangeMemberTitle {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).ChangeMemberTitleAndEmit(c.Body.GuildId, c.CharacterId, c.Body.TargetId, c.Body.Title, c.TransactionId)
	}
}

func handleCommandRequestDisband(db *gorm.DB) message.Handler[guild2.Command[guild2.RequestDisbandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.RequestDisbandBody]) {
		if c.Type != guild2.CommandTypeRequestDisband {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).RequestDisbandAndEmit(c.CharacterId, c.TransactionId)
	}
}

func handleCommandRequestCapacityIncrease(db *gorm.DB) message.Handler[guild2.Command[guild2.RequestCapacityIncreaseBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c guild2.Command[guild2.RequestCapacityIncreaseBody]) {
		if c.Type != guild2.CommandTypeRequestCapacityIncrease {
			return
		}

		_ = guild.NewProcessor(l, ctx, db).RequestCapacityIncreaseAndEmit(c.CharacterId, c.TransactionId)
	}
}
