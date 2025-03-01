package account

import (
	"atlas-account/configuration"
	"atlas-account/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

const (
	SystemError       = "SYSTEM_ERROR"
	NotRegistered     = "NOT_REGISTERED"
	DeletedOrBlocked  = "DELETED_OR_BLOCKED"
	AlreadyLoggedIn   = "ALREADY_LOGGED_IN"
	IncorrectPassword = "INCORRECT_PASSWORD"
	TooManyAttempts   = "TOO_MANY_ATTEMPTS"
)

type IdOperator func(tenant.Model, uint32) error

type IdProvider = func(uint32) model.Provider[Model]

type IdRetriever = func(uint32) (Model, error)

// GetById Retrieves a singular account by id.
var GetById = model.Compose(model.Curry(model.Compose[context.Context, IdProvider, IdRetriever])(model.Apply[uint32, Model]), ByIdProvider)

// ByIdProvider Retrieves a singular account by id.
var ByIdProvider = model.Flip(model.Compose(model.Curry(model.Compose[context.Context, tenant.Model, IdProvider]), byIdProvider))(tenant.MustFromContext)

var entityModelMapper = model.Map(modelFromEntity)
var entitySliceModelMapper = model.SliceMap(modelFromEntity)

func byIdProvider(db *gorm.DB) func(tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(tenant tenant.Model) func(id uint32) model.Provider[Model] {
		return func(id uint32) model.Provider[Model] {
			return model.Map(decorateState(tenant))(entityModelMapper(entityById(tenant, id)(db)))
		}
	}
}

type NameProvider = func(string) model.Provider[Model]

type NameRetriever = func(string) (Model, error)

// GetByName Retrieves a singular account by name.
var GetByName = model.Compose(model.Curry(model.Compose[context.Context, NameProvider, NameRetriever])(model.Apply[string, Model]), ByNameProvider)

// ByNameProvider Retrieves a singular account by name.
var ByNameProvider = model.Flip(model.Compose(model.Curry(model.Compose[context.Context, tenant.Model, NameProvider]), byNameProvider))(tenant.MustFromContext)

func byNameProvider(db *gorm.DB) func(t tenant.Model) func(string) model.Provider[Model] {
	return func(t tenant.Model) func(string) model.Provider[Model] {
		return func(name string) model.Provider[Model] {
			return model.Map(decorateState(t))(model.FirstProvider(entitySliceModelMapper(entitiesByName(t, name)(db))(model.ParallelMap()), model.Filters[Model]()))
		}
	}
}

type TenantProvider = func(context.Context) model.Provider[[]Model]

type TenantRetriever = func(context.Context) ([]Model, error)

// GetByTenant Retrieves all accounts in tenant.
var GetByTenant = model.Curry(model.Compose[*gorm.DB, TenantProvider, TenantRetriever])(model.Apply[context.Context, []Model])(ByTenantProvider)

// ByTenantProvider Retrieves all accounts in tenant.
var ByTenantProvider = model.Flip(model.Compose(model.Flip(byTenantProvider), tenant.MustFromContext))

func byTenantProvider(db *gorm.DB) func(tenant tenant.Model) model.Provider[[]Model] {
	return func(t tenant.Model) model.Provider[[]Model] {
		return model.SliceMap(decorateState(t))(entitySliceModelMapper(allInTenant(t)(db))(model.ParallelMap()))(model.ParallelMap())
	}
}

// FilterLoggedIn Filters results of provided provider by those who are logged in.
var FilterLoggedIn = model.Flip(model.Curry(model.FilteredProvider[Model]))(model.Filters(LoggedIn))

// LoggedInTenantProvider Retrieves all accounts logged into tenant.
var LoggedInTenantProvider = model.Compose(model.Curry(model.Compose[context.Context, model.Provider[[]Model], model.Provider[[]Model]])(FilterLoggedIn), ByTenantProvider)

// allTenants Retrieves all tenants with accounts associated.
var allTenants = model.FixedProvider(Get().Tenants())

func decorateState(tenant tenant.Model) model.Transformer[Model, Model] {
	return func(m Model) (Model, error) {
		st := Get().MaximalState(AccountKey{Tenant: tenant, AccountId: m.Id()})
		m.state = st
		return m, nil
	}
}

func GetInTransition(timeout time.Duration) ([]AccountKey, error) {
	return model.FixedProvider(Get().GetExpiredInTransition(timeout))()
}

func GetOrCreate(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(name string, password string, automaticRegister bool) (Model, error) {
	return func(name string, password string, automaticRegister bool) (Model, error) {
		m, err := GetByName(db)(ctx)(name)
		if err == nil {
			return m, nil
		}

		if !automaticRegister {
			l.Errorf("Unable to locate account by name [%s], and automatic account creation is not enabled.", name)
			return Model{}, errors.New("account not found")
		}

		return Create(l, db, ctx)(name, password)
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(name string, password string) (Model, error) {
	return func(name string, password string) (Model, error) {
		l.Debugf("Attempting to create account [%s] with password [%s].", name, password)
		hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			l.WithError(err).Errorf("Error generating hash when creating account [%s].", name)
			return Model{}, err
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			l.WithError(err).Errorf("Unable to locate tenant.")
			return Model{}, err
		}

		gender := byte(0)
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			gender = byte(10)
		}
		l.Debugf("Defaulting gender to [%d]. 0 = Male, 1 = Female, 10 = UI Choose. This is determined by Region and Version capabilities.", gender)

		m, err := create(db)(t, name, string(hashPass), gender)
		if err != nil {
			l.WithError(err).Errorf("Unable to create account [%s].", name)
			return Model{}, err
		}
		l.Debugf("Created account [%d] for [%s].", m.Id(), m.Name())
		_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(createdEventProvider()(m.Id(), name))
		return m, nil
	}
}

func Update(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(accountId uint32, input Model) (Model, error) {
	return func(accountId uint32, input Model) (Model, error) {
		a, err := GetById(db)(ctx)(accountId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate account being updated.")
			return Model{}, err
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			l.WithError(err).Errorf("Unable to locate tenant.")
			return Model{}, err
		}

		var modifiers = make([]EntityUpdateFunction, 0)

		if a.pin != input.pin && input.pin != "" {
			l.Debugf("Updating PIN [%s] of account [%d].", input.pin, accountId)
			modifiers = append(modifiers, updatePin(input.pin))
		}
		if a.pic != input.pic && input.pic != "" {
			l.Debugf("Updating PIC [%s] of account [%d].", input.pic, accountId)
			modifiers = append(modifiers, updatePic(input.pic))
		}
		if a.tos != input.tos && input.tos != false {
			l.Debugf("Updating TOS [%t] of account [%d].", input.tos, accountId)
			modifiers = append(modifiers, updateTos(input.tos))
		}
		if a.gender != input.gender {
			l.Debugf("Updating Gender [%d] of account [%d].", input.gender, accountId)
			modifiers = append(modifiers, updateGender(input.gender))
		}

		if len(modifiers) == 0 {
			return a, nil
		}

		err = update(db)(modifiers...)(t, accountId)
		if err != nil {
			l.WithError(err).Errorf("Unable to update account.")
			return Model{}, err
		}

		return GetById(db)(ctx)(accountId)
	}
}

func Login(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(sessionId uuid.UUID, accountId uint32, issuer string) error {
	t := tenant.MustFromContext(ctx)
	return func(sessionId uuid.UUID, accountId uint32, issuer string) error {
		a, err := GetById(db)(ctx)(accountId)
		if err != nil {
			return err
		}

		ak := AccountKey{Tenant: t, AccountId: accountId}
		sk := ServiceKey{SessionId: sessionId, Service: Service(issuer)}
		err = Get().Login(ak, sk)
		if err == nil {
			l.Debugf("State transition triggered a login.")
			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(loggedInEventProvider()(a.Id(), a.Name()))
			return err
		}
		return err
	}
}

func Logout(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(sessionId uuid.UUID, accountId uint32, issuer string) error {
	t := tenant.MustFromContext(ctx)
	return func(sessionId uuid.UUID, accountId uint32, issuer string) error {
		a, err := GetById(db)(ctx)(accountId)
		if err != nil {
			return err
		}

		if sessionId == uuid.Nil {
			ok := Get().Terminate(AccountKey{Tenant: t, AccountId: accountId})
			if !ok {
				return errors.New("error while logging out")
			}
		} else {
			ok := Get().Logout(AccountKey{Tenant: t, AccountId: accountId}, ServiceKey{SessionId: sessionId, Service: Service(issuer)})
			if !ok {
				return errors.New("error while logging out")
			}
		}
		l.Debugf("Logging out [%d] for [%s] via session [%s].", accountId, issuer, sessionId.String())
		return sendLogoutEvent(l)(ctx)("state transition")(a)
	}
}

func Teardown(l logrus.FieldLogger, db *gorm.DB) func() {
	return func() {
		sctx, span := otel.GetTracerProvider().Tracer("atlas-account").Start(context.Background(), "teardown")
		defer span.End()

		err := model.ForEachSlice(model.SliceMap(model.Always(model.Curry(tenant.WithContext)(sctx)))(allTenants)(model.ParallelMap()), teardownTenant(l)(db))
		if err != nil {
			l.WithError(err).Errorf("Error tearing down ")
		}
	}
}

func teardownTenant(l logrus.FieldLogger) func(db *gorm.DB) model.Operator[context.Context] {
	return func(db *gorm.DB) model.Operator[context.Context] {
		return func(ctx context.Context) error {
			logoutForModel := sendLogoutEvent(l)(ctx)("service shutdown")
			err := model.ForEachSlice(LoggedInTenantProvider(db)(ctx), logoutForModel, model.ParallelExecute())
			if err != nil {
				l.WithError(err).Errorf("Error destroying all monsters on teardown.")
			}
			return nil
		}
	}
}

func sendLogoutEvent(l logrus.FieldLogger) func(ctx context.Context) func(trigger string) model.Operator[Model] {
	return func(ctx context.Context) func(trigger string) model.Operator[Model] {
		return func(trigger string) model.Operator[Model] {
			return func(a Model) error {
				l.Debugf("Logging out [%d] [%s]. Triggered by [%s].", a.Id(), a.Name(), trigger)
				return producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(loggedOutEventProvider()(a.Id(), a.Name()))
			}
		}
	}
}

func AttemptLogin(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, name string, password string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, name string, password string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(sessionId uuid.UUID, name string, password string) error {
			return func(sessionId uuid.UUID, name string, password string) error {
				l.Debugf("Attemting login for [%s].", name)
				if checkLoginAttempts(sessionId) > 4 {
					l.Warnf("Session [%s] has attempted to log into (or create) an account too many times.", sessionId.String())
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, 0, TooManyAttempts))
					return nil
				}

				c, err := configuration.Get()
				if err != nil {
					l.WithError(err).Errorf("Error reading needed configuration.")
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, 0, SystemError))
					return err
				}

				a, err := GetOrCreate(l, db, ctx)(name, password, c.AutomaticRegister)
				if err != nil && !c.AutomaticRegister {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, 0, NotRegistered))
					return nil
				}
				if err != nil {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, 0, SystemError))
					return err
				}

				if a.Banned() {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), DeletedOrBlocked))
					return nil
				}

				// TODO implement ip, mac, and temporary banning practices

				if a.State() != StateNotLoggedIn {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), AlreadyLoggedIn))
					return nil
				} else if a.Password()[0] == uint8('$') && a.Password()[1] == uint8('2') && bcrypt.CompareHashAndPassword([]byte(a.Password()), []byte(password)) == nil {
					// TODO implement tos tracking
				} else {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), IncorrectPassword))
					return nil
				}

				err = Login(l, db, ctx)(sessionId, a.Id(), ServiceLogin)
				if err != nil {
					l.WithError(err).Errorf("Unable to record login.")
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
				}

				l.Debugf("Login successful for [%s].", name)

				if !a.TOS() && t.Region() != "JMS" {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(requestLicenseAgreementStatusProvider(sessionId, a.Id()))
					return nil
				}
				return producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(createdStatusProvider(sessionId, a.Id()))
			}
		}
	}
}

func ProgressState(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
	return func(ctx context.Context) func(db *gorm.DB) func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
			return func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
				a, err := GetById(db)(ctx)(accountId)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate account a session is being created for.")
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), NotRegistered))
					return err
				}

				l.Debugf("Received request to progress state for account [%d] to state [%d] from state [%d].", accountId, state, a.State())
				for k, v := range Get().GetStates(AccountKey{Tenant: t, AccountId: accountId}) {
					l.Debugf("Has state [%d] for [%s] via session [%s].", v.State, k.Service, k.SessionId.String())
				}
				if a.State() == StateNotLoggedIn {
					_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
					return errors.New("not logged in")
				}
				if state == StateNotLoggedIn {
					err = Logout(l, db, ctx)(sessionId, accountId, issuer)
					if err != nil {
						l.WithError(err).Errorf("Unable to logout account.")
						_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
						return err
					}
					return producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(stateChangedStatusProvider(sessionId, a.Id(), StateNotLoggedIn, params))
				}
				if state == StateLoggedIn {
					err = Login(l, db, ctx)(sessionId, accountId, issuer)
					if err != nil {
						l.WithError(err).Errorf("Unable to login account.")
						_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, a.Id(), SystemError))
						return err
					}
					return producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(stateChangedStatusProvider(sessionId, a.Id(), StateLoggedIn, params))
				}
				if state == StateTransition {
					err = Get().Transition(AccountKey{Tenant: t, AccountId: accountId}, ServiceKey{SessionId: sessionId, Service: Service(issuer)})
					if err == nil {
						l.Debugf("State transition triggered a transition.")
					}
					return producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(stateChangedStatusProvider(sessionId, a.Id(), StateTransition, params))
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventSessionStatusTopic)(errorStatusProvider(sessionId, 0, SystemError))
				return errors.New("invalid state")
			}
		}
	}
}

func checkLoginAttempts(sessionId uuid.UUID) byte {
	return 0
}
