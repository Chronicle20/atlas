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

const consumerCommand = "guild_command"
const consumerStatusEvent = "guild_status_event"

func CommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCommand)(EnvCommandTopic)(groupId)
	}
}

func RequestNameRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, c command[requestNameBody]) {
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
		}))
	}
}

func RequestEmblemRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, c command[requestEmblemBody]) {
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
		}))
	}
}

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvStatusEventTopic)(groupId)
	}
}

func CreatedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
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
		}))
	}
}

func DisbandedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventDisbandedBody]) {
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
		}))
	}
}

func RequestAgreementStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventRequestAgreementBody]) {
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
		}))
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

func EmblemUpdateStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventEmblemUpdatedBody]) {
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
		}))
	}
}

func MemberStatusUpdatedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberStatusUpdatedBody]) {
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
		}))
	}
}

func MemberTitleUpdatedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberTitleUpdatedBody]) {
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
		}))
	}
}

func NoticeUpdateStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventNoticeUpdatedBody]) {
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
		}))
	}
}

func MemberLeftStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberLeftBody]) {
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
		}))
	}
}

func MemberJoinedStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventMemberJoinedBody]) {
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
		}))
	}
}

func TitlesUpdateStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventTitlesUpdatedBody]) {
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
		}))
	}
}

func ErrorStatusEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		top, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return top, message.AdaptHandler(message.PersistentConfig(func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventErrorBody]) {
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
		}))
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
