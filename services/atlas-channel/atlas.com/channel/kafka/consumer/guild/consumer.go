package guild

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
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
			rf(consumer2.NewConfig(l)("guild_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("guild_status_event")(EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestName(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestEmblem(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDisbanded(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestAgreement(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEmblemUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMemberStatusUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMemberTitleUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleNoticeUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCapacityUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMemberLeft(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMemberJoined(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleTitlesUpdated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
			}
		}
	}
}

func handleError(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventErrorBody]) {
		if e.Type != StatusEventTypeError {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(e.Body.Error))
		if err != nil {
			l.Debugf("Unable to issue character [%d] guild error [%s].", e.Body.ActorId, err)
		}
	}
}

func announceGuildError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
			return func(errCode string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildErrorBody(l)(errCode))
			}
		}
	}
}

func handleTitlesUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventTitlesUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventTitlesUpdatedBody]) {
		if e.Type != StatusEventTypeTitlesUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceTitlesUpdated(l)(ctx)(wp)(e.GuildId, e.Body.Titles))
		if err != nil {
			l.Debugf("Unable to announce title update to [%d] guild.", e.GuildId)
		}
	}
}

func announceTitlesUpdated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, titles []string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, titles []string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, titles []string) model.Operator[session.Model] {
			return func(guildId uint32, titles []string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildTitleChangedBody(l)(guildId, titles))
			}
		}
	}
}

func handleMemberJoined(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberJoinedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberJoinedBody]) {
		if e.Type != StatusEventTypeMemberJoined {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] has joined.", e.GuildId, e.Body.CharacterId)
			return
		}

		// Inform members that guild member joined.
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline, guild.NotMember(e.Body.CharacterId))), announceMemberJoined(l)(ctx)(wp)(e.GuildId, e.Body.CharacterId, e.Body.Name, e.Body.JobId, e.Body.Level, e.Body.Title, e.Body.Online, e.Body.AllianceTitle))
		if err != nil {
			l.Debugf("Unable to announce character [%d] joined guild [%d] to current guild members.", e.Body.CharacterId, e.GuildId)
		}

		// Update character to show they are not in guild.
		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, announceGuildInfo(l)(ctx)(wp)(g))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] information to character [%d].", e.GuildId, e.Body.CharacterId)
		}

		// Update characters in map that x is in guild.
		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, _map.ForSessionsInSessionsMap(l)(ctx)(func(oid uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(e.Body.CharacterId, g)
		}))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] information to foreign characters.", e.GuildId)
		}
	}
}

func announceForeignGuildInfo(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, g guild.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, g guild.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, g guild.Model) model.Operator[session.Model] {
			return func(characterId uint32, g guild.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildNameChanged)(writer.ForeignGuildNameChangedBody(l)(characterId, g.Name()))(s)
					if err != nil {
						return err
					}
					err = session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(writer.ForeignGuildEmblemChangedBody(l)(characterId, g.Logo(), g.LogoColor(), g.LogoBackground(), g.LogoBackgroundColor()))(s)
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceGuildInfo(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(g guild.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(g guild.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(g guild.Model) model.Operator[session.Model] {
			return func(g guild.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildInfoBody(l, tenant.MustFromContext(ctx))(g))
			}
		}
	}
}

func announceMemberJoined(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
			return func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildMemberJoinedBody(l, tenant.MustFromContext(ctx))(guildId, characterId, name, jobId, level, title, online, allianceTitle))
			}
		}
	}
}

func handleMemberLeft(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberLeftBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberLeftBody]) {
		if e.Type != StatusEventTypeMemberLeft {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] has left.", e.GuildId, e.Body.CharacterId)
			return
		}

		var af model.Operator[session.Model]
		if e.Body.Force {
			af = announceMemberExpelled(l)(ctx)(wp)(e.GuildId, c.Id(), c.Name())
		} else {
			af = announceMemberLeft(l)(ctx)(wp)(e.GuildId, c.Id(), c.Name())
		}

		// Inform members that guild member left.
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline, guild.NotMember(e.Body.CharacterId))), af)
		if err != nil {
			l.Debugf("Unable to announce to guild [%d] that character [%d] has left.", e.GuildId, e.Body.CharacterId)
		}

		// Update character to show they are not in guild.
		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, announceGuildInfo(l)(ctx)(wp)(guild.Model{}))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce empty guild information to character [%d].", e.Body.CharacterId)
		}

		// Update characters in map that x is no longer in guild.
		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, _map.ForSessionsInSessionsMap(l)(ctx)(func(oid uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(e.Body.CharacterId, guild.Model{})
		}))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] information to foreign characters.", e.GuildId)
		}
	}
}

func announceMemberExpelled(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
			return func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildMemberExpelBody(l)(guildId, characterId, name))
			}
		}
	}
}

func announceMemberLeft(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
			return func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildMemberLeftBody(l)(guildId, characterId, name))
			}
		}
	}
}

func handleCapacityUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCapacityUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCapacityUpdatedBody]) {
		if e.Type != StatusEventTypeCapacityUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceCapacityChanged(l)(ctx)(wp)(e.GuildId, e.Body.Capacity))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce to guild [%d] members that the capacity has changed to [%d].", e.GuildId, e.Body.Capacity)
		}
	}
}

func announceCapacityChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, capacity uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, capacity uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, capacity uint32) model.Operator[session.Model] {
			return func(guildId uint32, capacity uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildCapacityChangedBody(l)(guildId, capacity))
			}
		}
	}
}

func handleNoticeUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventNoticeUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventNoticeUpdatedBody]) {
		if e.Type != StatusEventTypeNoticeUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceNoticeChanged(l)(ctx)(wp)(e.GuildId, e.Body.Notice))
		if err != nil {
			l.Debugf("Unable to guild [%d] members that the notice has changed to [%s].", e.GuildId, e.Body.Notice)
		}
	}
}

func announceNoticeChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, notice string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, notice string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, notice string) model.Operator[session.Model] {
			return func(guildId uint32, notice string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildNoticeChangedBody(l)(guildId, notice))
			}
		}
	}
}

func handleMemberTitleUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberTitleUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberTitleUpdatedBody]) {
		if e.Type != StatusEventTypeMemberTitleUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to issue guild [%d] member [%d] title [%d].", e.GuildId, e.Body.CharacterId, e.Body.Title)
			return
		}

		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceMemberTitleChanged(l)(ctx)(wp)(g, e.Body.CharacterId, e.Body.Title))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue guild [%d] member [%d] title [%d].", e.GuildId, e.Body.CharacterId, e.Body.Title)
		}
	}
}

func announceMemberTitleChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
			return func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					if s.CharacterId() != characterId {
						return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildMemberTitleUpdatedBody(l)(g.Id(), characterId, title))(s)
					} else {
						return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildInfoBody(l, t)(g))(s)
					}
				}
			}
		}
	}
}

func handleMemberStatusUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberStatusUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberStatusUpdatedBody]) {
		if e.Type != StatusEventTypeMemberStatusUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] status update to [%t].", e.GuildId, e.Body.CharacterId, e.Body.Online)
			return
		}

		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceMemberStatusUpdated(l)(ctx)(wp)(g, e.Body.CharacterId, e.Body.Online))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] status update to [%t].", e.GuildId, e.Body.CharacterId, e.Body.Online)
		}
	}
}

func announceMemberStatusUpdated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
			return func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					if s.CharacterId() != characterId {
						return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildMemberStatusUpdatedBody(l)(g.Id(), characterId, online))(s)
					} else {
						return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildInfoBody(l, t)(g))(s)
					}
				}
			}
		}
	}
}

func handleEmblemUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventEmblemUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventEmblemUpdatedBody]) {
		if e.Type != StatusEventTypeEmblemUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		// Inform members that the emblem changed.
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceEmblemChanged(l)(ctx)(wp)(g.Id(), e.Body.Logo, e.Body.LogoColor, e.Body.LogoBackground, e.Body.LogoBackgroundColor))
		if err != nil {
			l.Debugf("Unable to announce to guild [%d] members the emblem has changed.", e.GuildId)
		}

		// Inform foreign characters that the members emblem has changed.
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), _map.ForSessionsInSessionsMap(l)(ctx)(func(memberId uint32) model.Operator[session.Model] {
			return announceForeignEmblemChanged(l)(ctx)(wp)(memberId, e.Body.Logo, e.Body.LogoColor, e.Body.LogoBackground, e.Body.LogoBackgroundColor)
		}))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed to foreign characters in guild members maps.", e.GuildId)
		}
	}
}

func announceForeignEmblemChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
			return func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(writer.ForeignGuildEmblemChangedBody(l)(memberId, logo, logoColor, logoBackground, logoBackgroundColor))
			}
		}
	}
}

func announceEmblemChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
			return func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildEmblemChangedBody(l)(guildId, logo, logoColor, logoBackground, logoBackgroundColor))
			}
		}
	}
}

func handleRequestAgreement(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventRequestAgreementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventRequestAgreementBody]) {
		if e.Type != StatusEventTypeRequestAgreement {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := party.GetByMemberId(l)(ctx)(e.Body.ActorId)
		if err != nil {
			err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(writer.GuildOperationCreateError))
			if err != nil {
				l.Debugf("Unable to issue character [%d] guild error [%s].", e.Body.ActorId, err)
			}
			return
		}
		imf := party.OtherMemberInMap(sc.WorldId(), sc.ChannelId(), p.Leader().MapId(), p.LeaderId())
		mip := party.FilteredMemberProvider(imf)(model.FixedProvider(p))
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(party.MemberToMemberIdMapper(mip), requestGuildNameAgreement(l)(ctx)(wp)(p.Id(), p.LeaderName(), e.Body.ProposedName))
		if err != nil {
			l.Debugf("Unable to announce to party members that the guild [%s] is being created.", e.Body.ProposedName)
		}
	}
}

func requestGuildNameAgreement(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
			return func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildRequestAgreement(l)(partyId, leaderName, guildName))
			}
		}
	}
}

func handleDisbanded(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventDisbandedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventDisbandedBody]) {
		if e.Type != StatusEventTypeDisbanded {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		// Inform foreign characters that guild was left.
		_ = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), _map.ForSessionsInSessionsMap(l)(ctx)(func(memberId uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(memberId, guild.Model{})
		}))

		// Inform members that guild was disbanded.
		err := session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceGuildDisband(l)(ctx)(wp)(e.GuildId))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce to guild [%d] members that it has disbanded.", e.GuildId)
		}

		// Write empty guild information to character.
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceGuildInfo(l)(ctx)(wp)(guild.Model{}))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce empty guild information to former guild members.")
		}
	}
}

func announceGuildDisband(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32) model.Operator[session.Model] {
			return func(guildId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.GuildDisbandBody(l)(guildId))
			}
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
		if e.Type != StatusEventTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		// Inform foreign characters that guild was joined.
		_ = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), _map.ForSessionsInSessionsMap(l)(ctx)(func(memberId uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(memberId, g)
		}))

		// Write guild information to character.
		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceGuildInfo(l)(ctx)(wp)(g))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] information to current guild members.", g.Id())
		}
	}
}

func handleRequestEmblem(sc server.Model, wp writer.Producer) message.Handler[command[requestEmblemBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestEmblemBody]) {
		if c.Type != CommandTypeRequestEmblem {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(c.Body.WorldId), channel.Id(c.Body.ChannelId)) {
			return
		}

		err := session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, announceGuildEmblemRequest(l)(ctx)(wp))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce to character [%d] guild emblem request.", c.CharacterId)
		}
	}
}

func announceGuildEmblemRequest(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.RequestGuildEmblemBody(l))
		}
	}
}

func handleRequestName(sc server.Model, wp writer.Producer) message.Handler[command[requestNameBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestNameBody]) {
		if c.Type != CommandTypeRequestName {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(c.Body.WorldId), channel.Id(c.Body.ChannelId)) {
			return
		}

		err := session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, announceGuildNameRequest(l)(ctx)(wp))
		if err != nil {
			l.Debugf("Unable to request character [%d] input guild name.", c.CharacterId)
		}
	}
}

func announceGuildNameRequest(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			return session.Announce(l)(ctx)(wp)(writer.GuildOperation)(writer.RequestGuildNameBody(l))
		}
	}
}
