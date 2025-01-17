package session

import (
	"atlas-account/account"
	"atlas-account/configuration"
	"atlas-account/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	SystemError       = "SYSTEM_ERROR"
	NotRegistered     = "NOT_REGISTERED"
	DeletedOrBlocked  = "DELETED_OR_BLOCKED"
	AlreadyLoggedIn   = "ALREADY_LOGGED_IN"
	IncorrectPassword = "INCORRECT_PASSWORD"
	TooManyAttempts   = "TOO_MANY_ATTEMPTS"
)

func AttemptLogin(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, name string, password string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, name string, password string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(sessionId uuid.UUID, name string, password string) error {
			return func(sessionId uuid.UUID, name string, password string) error {
				l.Debugf("Attemting login for [%s].", name)
				if checkLoginAttempts(sessionId) > 4 {
					l.Warnf("Session [%s] has attempted to log into (or create) an account too many times.", sessionId.String())
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, 0, TooManyAttempts))
					return nil
				}

				c, err := configuration.Get()
				if err != nil {
					l.WithError(err).Errorf("Error reading needed configuration.")
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, 0, SystemError))
					return err
				}

				a, err := account.GetOrCreate(l, db, ctx)(name, password, c.AutomaticRegister)
				if err != nil && !c.AutomaticRegister {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, 0, NotRegistered))
					return nil
				}
				if err != nil {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, 0, SystemError))
					return err
				}

				if a.Banned() {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), DeletedOrBlocked))
					return nil
				}

				// TODO implement ip, mac, and temporary banning practices

				if a.State() != account.StateNotLoggedIn {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), AlreadyLoggedIn))
					return nil
				} else if a.Password()[0] == uint8('$') && a.Password()[1] == uint8('2') && bcrypt.CompareHashAndPassword([]byte(a.Password()), []byte(password)) == nil {
					// TODO implement tos tracking
				} else {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), IncorrectPassword))
					return nil
				}

				err = account.Login(l, db, ctx)(sessionId, a.Id(), account.ServiceLogin)
				if err != nil {
					l.WithError(err).Errorf("Unable to record login.")
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
				}

				l.Debugf("Login successful for [%s].", name)

				if !a.TOS() && t.Region() != "JMS" {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(requestLicenseAgreementStatusProvider(sessionId, a.Id()))
					return nil
				}
				return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdStatusProvider(sessionId, a.Id()))
			}
		}
	}
}

func ProgressState(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, issuer string, accountId uint32, state account.State, params interface{}) error {
	return func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, issuer string, accountId uint32, state account.State, params interface{}) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(sessionId uuid.UUID, issuer string, accountId uint32, state account.State, params interface{}) error {
			return func(sessionId uuid.UUID, issuer string, accountId uint32, state account.State, params interface{}) error {
				a, err := account.GetById(db)(ctx)(accountId)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate account a session is being created for.")
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), NotRegistered))
					return err
				}

				l.Debugf("Received request to progress state for account [%d] to state [%d] from state [%d].", accountId, state, a.State())
				for k, v := range account.Get().GetStates(account.AccountKey{Tenant: t, AccountId: accountId}) {
					l.Debugf("Has state [%d] for [%s] via session [%s].", v.State, k.Service, k.SessionId.String())
				}
				if a.State() == account.StateNotLoggedIn {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
					return errors.New("not logged in")
				}
				if state == account.StateNotLoggedIn {
					err = account.Logout(l, db, ctx)(sessionId, accountId, issuer)
					if err != nil {
						l.WithError(err).Errorf("Unable to logout account.")
						_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
						return err
					}
					return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(stateChangedStatusProvider(sessionId, a.Id(), account.StateNotLoggedIn, params))
				}
				if state == account.StateLoggedIn {
					err = account.Login(l, db, ctx)(sessionId, accountId, issuer)
					if err != nil {
						l.WithError(err).Errorf("Unable to login account.")
						_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
						return err
					}
					return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(stateChangedStatusProvider(sessionId, a.Id(), account.StateLoggedIn, params))
				}
				if state == account.StateTransition {
					err = account.Get().Transition(account.AccountKey{Tenant: t, AccountId: accountId}, account.ServiceKey{SessionId: sessionId, Service: account.Service(issuer)})
					if err == nil {
						l.Debugf("State transition triggered a transition.")
					}
					return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(stateChangedStatusProvider(sessionId, a.Id(), account.StateTransition, params))
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorStatusProvider(sessionId, 0, SystemError))
				return errors.New("invalid state")
			}
		}
	}
}

func checkLoginAttempts(sessionId uuid.UUID) byte {
	return 0
}
