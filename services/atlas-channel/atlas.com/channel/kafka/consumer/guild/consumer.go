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
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("guild_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("guild_status_event")(EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
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

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(e.Body.Error))
	}
}

func announceGuildError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
			guildOperationFunc := session.Announce(l)(ctx)(wp)(writer.GuildOperation)
			return func(errCode string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := guildOperationFunc(s, writer.GuildErrorBody(l)(errCode))
					if err != nil {
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				}
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

		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceTitlesUpdated(l)(ctx)(wp)(e.GuildId, e.Body.Titles))
	}
}

func announceTitlesUpdated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, titles []string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, titles []string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, titles []string) model.Operator[session.Model] {
			return func(guildId uint32, titles []string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildTitleChangedBody(l)(guildId, titles))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] guild titles update.", s.CharacterId())
						return err
					}
					return nil
				}
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
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline, guild.NotMember(e.Body.CharacterId))), announceMemberJoined(l)(ctx)(wp)(e.GuildId, e.Body.CharacterId, e.Body.Name, e.Body.JobId, e.Body.Level, e.Body.Title, e.Body.Online, e.Body.AllianceTitle))

		// Update character to show they are not in guild.
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, announceGuildInfo(l)(ctx)(wp)(g))

		// Update characters in map that x is in guild.
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, _map.ForSessionsInSessionsMap(l)(ctx)(func(oid uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(e.Body.CharacterId, g)
		}))
	}
}

func announceForeignGuildInfo(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, g guild.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, g guild.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, g guild.Model) model.Operator[session.Model] {
			return func(characterId uint32, g guild.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildNameChanged)(s, writer.ForeignGuildNameChangedBody(l)(characterId, g.Name()))
					if err != nil {
						return err
					}
					err = session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(s, writer.ForeignGuildEmblemChangedBody(l)(characterId, g.Logo(), g.LogoColor(), g.LogoBackground(), g.LogoBackgroundColor()))
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
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(g guild.Model) model.Operator[session.Model] {
			return func(g guild.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce guild [%d] information to character [%d].", g.Id(), s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceMemberJoined(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
			return func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberJoinedBody(l, t)(guildId, characterId, name, jobId, level, title, online, allianceTitle))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				}
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

		c, err := character.GetById(l)(ctx)()(e.Body.CharacterId)
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
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline, guild.NotMember(e.Body.CharacterId))), af)

		// Update character to show they are not in guild.
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, announceGuildInfo(l)(ctx)(wp)(guild.Model{}))

		// Update characters in map that x is no longer in guild.
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, _map.ForSessionsInSessionsMap(l)(ctx)(func(oid uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(e.Body.CharacterId, guild.Model{})
		}))
	}
}

func announceMemberExpelled(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
			return func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberExpelBody(l)(guildId, characterId, name))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that [%d] has been expelled from the guild [%d].", s.CharacterId(), characterId, guildId)
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceMemberLeft(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
			return func(guildId uint32, characterId uint32, name string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberLeftBody(l)(guildId, characterId, name))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that [%d] has left the guild [%d].", s.CharacterId(), characterId, guildId)
						return err
					}
					return nil
				}
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

		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceCapacityChanged(l)(ctx)(wp)(e.GuildId, e.Body.Capacity))
	}
}

func announceCapacityChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, capacity uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, capacity uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, capacity uint32) model.Operator[session.Model] {
			return func(guildId uint32, capacity uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildCapacityChangedBody(l)(guildId, capacity))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that the guild [%d] capacity has changed to [%d].", s.CharacterId(), guildId, capacity)
						return err
					}
					return nil
				}
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

		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceNoticeChanged(l)(ctx)(wp)(e.GuildId, e.Body.Notice))
	}
}

func announceNoticeChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, notice string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, notice string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, notice string) model.Operator[session.Model] {
			return func(guildId uint32, notice string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildNoticeChangedBody(l)(guildId, notice))
					if err != nil {
						l.Debugf("Unable to character [%d] that the guild [%d] notice has changed to [%s].", s.CharacterId(), guildId, notice)
						return err
					}
					return nil
				}
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
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceMemberTitleChanged(l)(ctx)(wp)(g, e.Body.CharacterId, e.Body.Title))
	}
}

func announceMemberTitleChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
			return func(g guild.Model, characterId uint32, title byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					if s.CharacterId() != characterId {
						err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberTitleUpdatedBody(l)(g.Id(), characterId, title))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] title [%d].", s.CharacterId(), g.Id(), characterId, title)
							return err
						}
						return nil
					} else {
						// Refresh own list, as the character will appear offline if not.
						err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] title [%d].", s.CharacterId(), g.Id(), characterId, title)
							return err
						}
						return nil
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
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceMemberStatusUpdated(l)(ctx)(wp)(g, e.Body.CharacterId, e.Body.Online))
	}
}

func announceMemberStatusUpdated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
			return func(g guild.Model, characterId uint32, online bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					if s.CharacterId() != characterId {
						err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberStatusUpdatedBody(l)(g.Id(), characterId, online))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] online [%t].", s.CharacterId(), g.Id(), characterId, online)
							return err
						}
						return nil
					} else {
						// Refresh own list, as the character will appear offline if not.
						err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] online [%t].", s.CharacterId(), g.Id(), characterId, online)
							return err
						}
						return nil
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
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceEmblemChanged(l)(ctx)(wp)(g.Id(), e.Body.Logo, e.Body.LogoColor, e.Body.LogoBackground, e.Body.LogoBackgroundColor))

		// Inform foreign characters that the members emblem has changed.
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), _map.ForSessionsInSessionsMap(l)(ctx)(func(memberId uint32) model.Operator[session.Model] {
			return announceForeignEmblemChanged(l)(ctx)(wp)(memberId, e.Body.Logo, e.Body.LogoColor, e.Body.LogoBackground, e.Body.LogoBackgroundColor)
		}))
	}
}

func announceForeignEmblemChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
			return func(memberId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(s, writer.ForeignGuildEmblemChangedBody(l)(memberId, logo, logoColor, logoBackground, logoBackgroundColor))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that foreign character [%d] guild emblem has changed.", s.CharacterId(), memberId)
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceEmblemChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
			return func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildEmblemChangedBody(l)(guildId, logo, logoColor, logoBackground, logoBackgroundColor))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that the guild [%d] emblem has changed.", s.CharacterId(), guildId)
						return err
					}
					return nil
				}
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
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(writer.GuildOperationCreateError))
			return
		}
		_ = session.ForEachByCharacterId(sc.Tenant())(party.GetMemberIds(l)(ctx)(p.Id(), model.Filters[party.MemberModel]()), requestGuildNameAgreement(l)(ctx)(wp)(p.Id(), p.LeaderName(), e.Body.ProposedName))
	}
}

func requestGuildNameAgreement(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
			guildOperationFunc := session.Announce(l)(ctx)(wp)(writer.GuildOperation)
			return func(partyId uint32, leaderName string, guildName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := guildOperationFunc(s, writer.GuildRequestAgreement(l)(partyId, leaderName, guildName))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that the guild [%s] is being created.", s.CharacterId(), guildName)
						return err
					}
					return nil
				}
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
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), _map.ForSessionsInSessionsMap(l)(ctx)(func(memberId uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(memberId, guild.Model{})
		}))

		// Inform members that guild was disbanded.
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceGuildDisband(l)(ctx)(wp)(e.GuildId))

		// Write empty guild information to character.
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceGuildInfo(l)(ctx)(wp)(guild.Model{}))
	}
}

func announceGuildDisband(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(guildId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(guildId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(guildId uint32) model.Operator[session.Model] {
			return func(guildId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildDisbandBody(l)(guildId))
					if err != nil {
						l.Debugf("Unable to announce to character [%d] that the guild [%d] has disbanded.", s.CharacterId(), guildId)
						return err
					}
					return nil
				}
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
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), _map.ForSessionsInSessionsMap(l)(ctx)(func(memberId uint32) model.Operator[session.Model] {
			return announceForeignGuildInfo(l)(ctx)(wp)(memberId, g)
		}))

		// Write guild information to character.
		_ = session.ForEachByCharacterId(sc.Tenant())(guild.GetMemberIds(l)(ctx)(e.GuildId, model.Filters(guild.MemberOnline)), announceGuildInfo(l)(ctx)(wp)(g))
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

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, announceGuildEmblemRequest(l)(ctx)(wp))
	}
}

func announceGuildEmblemRequest(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			return func(s session.Model) error {
				err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.RequestGuildEmblemBody(l))
				if err != nil {
					l.Debugf("Unable to announce to character [%d] guild emblem request.", s.CharacterId())
					return err
				}
				return nil
			}
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

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, announceGuildNameRequest(l)(ctx)(wp))
	}
}

func announceGuildNameRequest(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			return func(s session.Model) error {
				err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.RequestGuildNameBody(l))
				if err != nil {
					l.Debugf("Unable to request character [%d] input guild name.", s.CharacterId())
					return err
				}
				return nil
			}
		}
	}
}
