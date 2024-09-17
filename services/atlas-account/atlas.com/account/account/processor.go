package account

import (
	"atlas-account/database"
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

type IdOperator func(tenant.Model, uint32) error

type IdProvider = func(uint32) model.Provider[Model]

type IdRetriever = func(uint32) (Model, error)

// GetById Retrieves a singular account by id.
var GetById = model.Compose(model.Curry(model.Compose[context.Context, IdProvider, IdRetriever])(model.Apply[uint32, Model]), ByIdProvider)

// ByIdProvider Retrieves a singular account by id.
var ByIdProvider = model.Flip(model.Compose(model.Curry(model.Compose[context.Context, tenant.Model, IdProvider]), byIdProvider))(tenant.MustFromContext)

func byIdProvider(db *gorm.DB) func(tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(tenant tenant.Model) func(id uint32) model.Provider[Model] {
		return func(id uint32) model.Provider[Model] {
			return model.Map(database.ModelProvider[Model, entity](db)(entityById(tenant, id), modelFromEntity), decorateState(tenant))
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
			return model.Map(model.FirstProvider(database.ModelSliceProvider[Model, entity](db)(entitiesByName(t, name), modelFromEntity), model.Filters[Model]()), decorateState(t))
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
		return model.SliceMap(database.ModelSliceProvider[Model, entity](db)(allInTenant(t), modelFromEntity), decorateState(t))
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
		_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicAccountStatus)(createdEventProvider()(m.Id(), name))
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
	return func(sessionId uuid.UUID, accountId uint32, issuer string) error {
		a, err := GetById(db)(ctx)(accountId)
		if err != nil {
			return err
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			l.WithError(err).Errorf("Unable to locate tenant.")
			return err
		}

		ak := AccountKey{Tenant: t, AccountId: accountId}
		sk := ServiceKey{SessionId: sessionId, Service: Service(issuer)}
		err = Get().Login(ak, sk)
		if err == nil {
			l.Debugf("State transition triggered a login.")
			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicAccountStatus)(loggedInEventProvider()(a.Id(), a.Name()))
			return err
		}
		return err
	}
}

func Logout(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(sessionId uuid.UUID, accountId uint32, issuer string) error {
	return func(sessionId uuid.UUID, accountId uint32, issuer string) error {
		a, err := GetById(db)(ctx)(accountId)
		if err != nil {
			return err
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			l.WithError(err).Errorf("Unable to locate tenant.")
			return err
		}

		ok := Get().Logout(AccountKey{Tenant: t, AccountId: accountId}, ServiceKey{SessionId: sessionId, Service: Service(issuer)})
		if ok {
			return sendLogoutEvent(l)(ctx)("state transition")(a)
		}
		return errors.New("error while logging out")
	}
}

func Teardown(l logrus.FieldLogger, db *gorm.DB) func() {
	return func() {
		sctx, span := otel.GetTracerProvider().Tracer("atlas-account").Start(context.Background(), "teardown")
		defer span.End()

		err := model.ForEachSlice(model.SliceMap(allTenants, model.Always(model.Curry(tenant.WithContext)(sctx))), teardownTenant(l)(db))
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
				return producer.ProviderImpl(l)(ctx)(EnvEventTopicAccountStatus)(loggedOutEventProvider()(a.Id(), a.Name()))
			}
		}
	}
}
