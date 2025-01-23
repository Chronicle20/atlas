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

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(e.Body.Error))
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

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] title has changed.", e.GuildId)
			return
		}

		for _, gm := range g.Members() {
			go func() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
					err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildTitleChangedBody(l)(g.Id(), e.Body.Titles))
					if err != nil {
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				})
			}()
		}
	}
}

func handleMemberJoined(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberJoinedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberJoinedBody]) {
		if e.Type != StatusEventTypeMemberJoined {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		c, err := character.GetById(l)(ctx)(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] has joined.", e.GuildId, e.Body.CharacterId)
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] has joined.", e.GuildId, e.Body.CharacterId)
			return
		}

		// Inform members that guild member joined.
		go func() {
			for _, gm := range g.Members() {
				if gm.Online() && gm.CharacterId() != e.Body.CharacterId {
					session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
						err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberJoinedBody(l, t)(g.Id(), c.Id(), e.Body.Name, e.Body.JobId, e.Body.Level, e.Body.Title, e.Body.Online, e.Body.AllianceTitle))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
							return err
						}
						return nil
					})
				}
			}
		}()

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, func(s session.Model) error {
			// Update characters in map that x is no longer in guild.
			go func() {
				_ = _map.ForSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), s.MapId(), func(os session.Model) error {
					err = session.Announce(l)(ctx)(wp)(writer.GuildNameChanged)(os, writer.ForeignGuildNameChangedBody(l)(s.CharacterId(), g.Name()))
					if err != nil {
						return err
					}
					err = session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(os, writer.ForeignGuildEmblemChangedBody(l)(s.CharacterId(), g.Logo(), g.LogoColor(), g.LogoBackground(), g.LogoBackgroundColor()))
					if err != nil {
						return err
					}
					return nil
				})
			}()

			// Update character to show they are not in guild.
			go func() {
				err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
				if err != nil {
				}
			}()
			return nil
		})
	}
}

func handleMemberLeft(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberLeftBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberLeftBody]) {
		if e.Type != StatusEventTypeMemberLeft {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		c, err := character.GetById(l)(ctx)(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] has left.", e.GuildId, e.Body.CharacterId)
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] member [%d] has left.", e.GuildId, e.Body.CharacterId)
			return
		}

		// Inform members that guild member left.
		go func() {
			for _, gm := range g.Members() {
				if gm.Online() && gm.CharacterId() != e.Body.CharacterId {
					session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
						if e.Body.Force {
							err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberExpelBody(l)(g.Id(), c.Id(), c.Name()))
							if err != nil {
								l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
								return err
							}
							return nil
						} else {
							err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberLeftBody(l)(g.Id(), c.Id(), c.Name()))
							if err != nil {
								l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
								return err
							}
							return nil
						}
					})
				}
			}
		}()

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, func(s session.Model) error {
			// Update characters in map that x is no longer in guild.
			go func() {
				_ = _map.ForSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), s.MapId(), func(os session.Model) error {
					err = session.Announce(l)(ctx)(wp)(writer.GuildNameChanged)(os, writer.ForeignGuildNameChangedBody(l)(s.CharacterId(), ""))
					if err != nil {
						return err
					}
					return nil
				})
			}()

			// Update character to show they are not in guild.
			go func() {
				err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(guild.Model{}))
				if err != nil {
				}
			}()
			return nil
		})
	}
}

func handleCapacityUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCapacityUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCapacityUpdatedBody]) {
		if e.Type != StatusEventTypeCapacityUpdated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] capacity has changed.", e.GuildId)
			return
		}

		for _, gm := range g.Members() {
			go func() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
					err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildCapacityChangedBody(l)(g.Id(), e.Body.Capacity))
					if err != nil {
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				})
			}()
		}
	}
}

func handleNoticeUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventNoticeUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventNoticeUpdatedBody]) {
		if e.Type != StatusEventTypeNoticeUpdated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] notice has changed.", e.GuildId)
			return
		}

		for _, gm := range g.Members() {
			go func() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
					err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildNoticeChangedBody(l)(g.Id(), e.Body.Notice))
					if err != nil {
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
						return err
					}
					return nil
				})
			}()
		}
	}
}

func handleMemberTitleUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberTitleUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberTitleUpdatedBody]) {
		if e.Type != StatusEventTypeMemberTitleUpdated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		for _, gm := range g.Members() {
			go func() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
					if gm.CharacterId() != e.Body.CharacterId {
						err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberTitleUpdatedBody(l)(g.Id(), e.Body.CharacterId, e.Body.Title))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] title [%d].", s.CharacterId(), g.Id(), e.Body.CharacterId, e.Body.Title)
							return err
						}
						return nil
					} else {
						// Refresh own list, as the character will appear offline if not.
						err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] title [%d].", s.CharacterId(), g.Id(), e.Body.CharacterId, e.Body.Title)
							return err
						}
						return nil
					}
				})
			}()
		}
	}
}

func handleMemberStatusUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventMemberStatusUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberStatusUpdatedBody]) {
		if e.Type != StatusEventTypeMemberStatusUpdated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		for _, gm := range g.Members() {
			go func() {
				session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
					if gm.CharacterId() != e.Body.CharacterId {
						err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildMemberStatusUpdatedBody(l)(g.Id(), e.Body.CharacterId, e.Body.Online))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] online [%t].", s.CharacterId(), g.Id(), e.Body.CharacterId, e.Body.Online)
							return err
						}
						return nil
					} else {
						// Refresh own list, as the character will appear offline if not.
						err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild [%d] member [%d] online [%t].", s.CharacterId(), g.Id(), e.Body.CharacterId, e.Body.Online)
							return err
						}
						return nil
					}
				})
			}()
		}
	}
}

func handleEmblemUpdated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventEmblemUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventEmblemUpdatedBody]) {
		if e.Type != StatusEventTypeEmblemUpdated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		for _, gm := range g.Members() {
			if gm.Online() {
				go func() {
					session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
						err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildEmblemChangedBody(l)(g.Id(), e.Body.Logo, e.Body.LogoColor, e.Body.LogoBackground, e.Body.LogoBackgroundColor))
						if err != nil {
							l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
							return err
						}

						return _map.ForSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), s.MapId(), func(os session.Model) error {
							err = session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(os, writer.ForeignGuildEmblemChangedBody(l)(s.CharacterId(), e.Body.Logo, e.Body.LogoColor, e.Body.LogoBackground, e.Body.LogoBackgroundColor))
							if err != nil {
								l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
								return err
							}
							return nil
						})
					})
				}()
			}
		}
	}
}

func handleRequestAgreement(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventRequestAgreementBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventRequestAgreementBody]) {
		if e.Type != StatusEventTypeRequestAgreement {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		p, err := party.GetByMemberId(l)(ctx)(e.Body.ActorId)
		if err != nil {
			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.Body.ActorId, announceGuildError(l)(ctx)(wp)(writer.GuildOperationCreateError))
			return
		}

		leaderName := ""
		for _, m := range p.Members() {
			if p.LeaderId() == m.Id() {
				leaderName = m.Name()
				break
			}
		}

		for _, m := range p.Members() {
			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(m.Id(), requestGuildNameAgreement(l)(ctx)(wp)(p.Id(), leaderName, e.Body.ProposedName))
		}
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
						l.Debugf("Unable to issue character [%d] guild error [%s].", s.CharacterId(), err)
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

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		// Inform members that guild member left.
		for _, cid := range e.Body.Members {
			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(cid, func(s session.Model) error {
				go func() {
					_ = _map.ForSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), s.MapId(), func(os session.Model) error {
						err := session.Announce(l)(ctx)(wp)(writer.GuildNameChanged)(os, writer.ForeignGuildNameChangedBody(l)(s.CharacterId(), ""))
						if err != nil {
							return err
						}
						return nil
					})
				}()

				go func() {
					err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildDisbandBody(l)(e.GuildId))
					if err != nil {
					}

					err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(guild.Model{}))
					if err != nil {
					}
				}()
				return nil
			})
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
		if e.Type != StatusEventTypeCreated {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if sc.WorldId() != e.WorldId {
			return
		}

		g, err := guild.GetById(l)(ctx)(e.GuildId)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce guild [%d] emblem has changed.", e.GuildId)
			return
		}

		// Inform members that guild member left.
		for _, gm := range g.Members() {
			session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(gm.CharacterId(), func(s session.Model) error {
				go func() {
					_ = _map.ForSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), s.MapId(), func(os session.Model) error {
						err = session.Announce(l)(ctx)(wp)(writer.GuildNameChanged)(os, writer.ForeignGuildNameChangedBody(l)(s.CharacterId(), g.Name()))
						if err != nil {
							return err
						}
						err = session.Announce(l)(ctx)(wp)(writer.GuildEmblemChanged)(os, writer.ForeignGuildEmblemChangedBody(l)(s.CharacterId(), g.Logo(), g.LogoColor(), g.LogoBackground(), g.LogoBackgroundColor()))
						if err != nil {
							return err
						}
						return nil
					})
				}()

				go func() {
					err = session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.GuildInfoBody(l, t)(g))
					if err != nil {
					}
				}()
				return nil
			})
		}
	}
}

func handleRequestEmblem(sc server.Model, wp writer.Producer) message.Handler[command[requestEmblemBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestEmblemBody]) {
		if c.Type != CommandTypeRequestEmblem {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), c.Body.WorldId, c.Body.ChannelId) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.RequestGuildEmblemBody(l))
			if err != nil {
				l.Debugf("Unable to request character [%d] input guild emblem.", s.CharacterId())
				return err
			}
			return nil
		})
	}
}

func handleRequestName(sc server.Model, wp writer.Producer) message.Handler[command[requestNameBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestNameBody]) {
		if c.Type != CommandTypeRequestName {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), c.Body.WorldId, c.Body.ChannelId) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(c.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(writer.GuildOperation)(s, writer.RequestGuildNameBody(l))
			if err != nil {
				l.Debugf("Unable to request character [%d] input guild name.", s.CharacterId())
				return err
			}
			return nil
		})
	}
}
