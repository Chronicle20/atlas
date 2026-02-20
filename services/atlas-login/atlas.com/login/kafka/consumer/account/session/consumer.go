package session

import (
	"atlas-login/account"
	"atlas-login/configuration"
	consumer2 "atlas-login/kafka/consumer"
	session2 "atlas-login/kafka/message/account/session"
	"atlas-login/session"
	model2 "atlas-login/socket/model"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"sort"
	"time"

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
			rf(consumer2.NewConfig(l)("account_session_status_event")(session2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(tenant tenant.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(tenant tenant.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(session2.EnvEventStatusTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedAccountSessionStatusEvent(tenant, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLicenseAgreementAccountSessionStatusEvent(tenant, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorAccountSessionStatusEvent(tenant, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStateChangedAccountSessionStatusEvent(tenant, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}
func handleCreatedAccountSessionStatusEvent(t tenant.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.CreatedStatusEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.CreatedStatusEventBody]) {
		if e.Type != session2.EventStatusTypeCreated {
			return
		}

		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		sp := session.NewProcessor(l, ctx)
		sp.IfPresentById(e.SessionId, func(s session.Model) error {
			a, err := account.NewProcessor(l, ctx).GetById(e.AccountId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve account [%d] that had a session [%s] created for it.", e.AccountId, s.SessionId().String())
				return err
			}

			s = sp.SetAccountId(s.SessionId(), a.Id())
			_ = sp.SessionCreated(s)

			sc, err := configuration.GetTenantConfig(t.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to find server configuration.")
				return err
			}

			err = session.Announce(l)(wp)(writer.AuthSuccess)(s, writer.AuthSuccessBody(t)(a.Id(), a.Name(), a.Gender(), sc.UsesPin, a.PIC()))
			if err != nil {
				l.WithError(err).Errorf("Unable to show successful authorization for account %d", a.Id())
				return err
			}

			if t.Region() == "JMS" {
				_ = announceServerInformation(l)(ctx)(wp)(s)
			}

			return err
		})
	}
}

func handleLicenseAgreementAccountSessionStatusEvent(t tenant.Model, wp writer.Producer) message.Handler[session2.StatusEvent[any]] {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[any]) {
		if e.Type != session2.EventStatusTypeRequestLicenseAgreement {
			return
		}

		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		sp := session.NewProcessor(l, ctx)
		sp.IfPresentById(e.SessionId, func(s session.Model) error {
			a, err := account.NewProcessor(l, ctx).GetById(e.AccountId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve account [%d] that had a session [%s] created for it.", e.AccountId, s.SessionId().String())
				return err
			}

			s = sp.SetAccountId(s.SessionId(), a.Id())
			_ = sp.SessionCreated(s)

			return announceError(l)(ctx)(wp)("LICENSE_AGREEMENT")(s)
		})
	}
}

func handleErrorAccountSessionStatusEvent(t tenant.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.ErrorStatusEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.ErrorStatusEventBody]) {
		if e.Type != session2.EventStatusTypeError {
			return
		}

		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if e.Body.Code != session2.EventStatusErrorCodeDeletedOrBlocked {
			session.NewProcessor(l, ctx).IfPresentById(e.SessionId, announceError(l)(ctx)(wp)(e.Body.Code))
		} else if !e.Body.Until.IsZero() {
			session.NewProcessor(l, ctx).IfPresentById(e.SessionId, announceTemporaryBan(l)(ctx)(wp)(e.Body.Until, e.Body.Reason))
		} else {
			session.NewProcessor(l, ctx).IfPresentById(e.SessionId, announcePermanentBan(l)(ctx)(wp))
		}
	}
}

func announcePermanentBan(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) model.Operator[session.Model] {
			authPermanentBanFunc := session.Announce(l)(wp)(writer.AuthPermanentBan)
			return func(s session.Model) error {
				err := authPermanentBanFunc(s, writer.AuthPermanentBanBody(l, t))
				if err != nil {
					l.WithError(err).Errorf("Unable to show account is permanently banned.")
				}
				return err
			}
		}
	}
}

func announceTemporaryBan(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(until time.Time, reason byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(until time.Time, reason byte) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(until time.Time, reason byte) model.Operator[session.Model] {
			authTemporaryBanFunc := session.Announce(l)(wp)(writer.AuthTemporaryBan)
			return func(until time.Time, reason byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := authTemporaryBanFunc(s, writer.AuthTemporaryBanBody(l, t)(until, reason))
					if err != nil {
						l.WithError(err).Errorf("Unable to show account is temporary banned.")
					}
					return err
				}
			}
		}
	}
}

func announceError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(reason string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(reason string) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(reason string) model.Operator[session.Model] {
			authLoginFailedFunc := session.Announce(l)(wp)(writer.AuthLoginFailed)
			return func(reason string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := authLoginFailedFunc(s, writer.AuthLoginFailedBody(l, t)(reason))
					if err != nil {
						l.WithError(err).Errorf("Unable to issue [%s].", writer.AuthLoginFailed)
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceServerInformation(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		p := world.NewProcessor(l, ctx)
		ws, err := p.GetAll()
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve worlds to display to session.")
		}
		sort.Slice(ws, func(i, j int) bool {
			return ws[i].Id() < ws[j].Id()
		})

		if len(ws) == 0 {
			l.Warnf("Responding with no valid worlds.")
		}

		return func(wp writer.Producer) model.Operator[session.Model] {
			return model.ThenOperator(announceServerList(l)(ctx)(wp)(ws), model.Operators[session.Model](announceLastWorld(l)(ctx)(wp), announceRecommendedWorlds(l)(ctx)(wp)(ws)))
		}
	}
}

func announceRecommendedWorlds(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
			serverListRecommendationFunc := session.Announce(l)(wp)(writer.ServerListRecommendations)
			return func(ws []world.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					var rs = make([]model2.Recommendation, 0)
					for _, x := range ws {
						if x.Recommended() {
							rs = append(rs, model2.NewWorldRecommendation(x.Id(), x.RecommendedMessage()))
						}
					}
					err := serverListRecommendationFunc(s, writer.ServerListRecommendationsBody(l, ctx)(rs))
					if err != nil {
						l.WithError(err).Errorf("Unable to issue recommended worlds")
						return err
					}
					return nil
				}
			}
		}
	}
}

func announceLastWorld(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			selectWorldFunc := session.Announce(l)(wp)(writer.SelectWorld)
			return func(s session.Model) error {
				err := selectWorldFunc(s, writer.SelectWorldBody(0))
				if err != nil {
					l.WithError(err).Errorf("Unable to identify the last world a account was logged into")
					return err
				}
				return nil
			}
		}
	}
}

func announceServerList(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(ws []world.Model) model.Operator[session.Model] {
			serverListEntryFunc := session.Announce(l)(wp)(writer.ServerListEntry)
			serverListEndFunc := session.Announce(l)(wp)(writer.ServerListEnd)
			return func(ws []world.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					for _, x := range ws {
						var cls []model2.Load
						for _, c := range x.Channels() {
							cls = append(cls, model2.NewChannelLoad(c.ChannelId(), c.CurrentCapacity()))
						}

						err := serverListEntryFunc(s, writer.ServerListEntryBody(t)(x.Id(), x.Name(), x.State(), x.EventMessage(), cls))
						if err != nil {
							l.WithError(err).Errorf("Unable to write server list entry for [%d]", x.Id())
						}
					}
					err := serverListEndFunc(s, writer.ServerListEndBody)
					if err != nil {
						l.WithError(err).Errorf("Unable to complete writing the server list")
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleStateChangedAccountSessionStatusEvent(t tenant.Model, wp writer.Producer) message.Handler[session2.StatusEvent[session2.StateChangedEventBody[model2.ChannelSelect]]] {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.StateChangedEventBody[model2.ChannelSelect]]) {
		if e.Type != session2.EventStatusTypeStateChanged {
			return
		}

		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentById(e.SessionId, processStateReturn(l)(ctx)(wp)(e.AccountId, e.Body.State, e.Body.Params))
	}
}

func processStateReturn(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(accountId uint32, state uint8, params model2.ChannelSelect) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(accountId uint32, state uint8, params model2.ChannelSelect) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(accountId uint32, state uint8, params model2.ChannelSelect) model.Operator[session.Model] {
			serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
			return func(accountId uint32, state uint8, params model2.ChannelSelect) model.Operator[session.Model] {
				return func(s session.Model) error {
					if len(params.IPAddress) <= 0 {
						return nil
					}

					err := serverIpFunc(s, writer.ServerIPBody(l, t)(params.IPAddress, params.Port, params.CharacterId))
					if err != nil {
						l.WithError(err).Errorf("Unable to write server ip response due to error.")
						return err
					}
					return nil
				}
			}
		}
	}
}
