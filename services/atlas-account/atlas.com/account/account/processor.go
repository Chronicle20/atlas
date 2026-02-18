package account

import (
	"atlas-account/ban"
	"atlas-account/configuration"
	"atlas-account/kafka/message"
	account2 "atlas-account/kafka/message/account"
	ban2 "atlas-account/kafka/message/ban"
	"atlas-account/kafka/producer"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	ReasonCodeInvalidAttempts = byte(99)

	SystemError       = "SYSTEM_ERROR"
	NotRegistered     = "NOT_REGISTERED"
	DeletedOrBlocked  = "DELETED_OR_BLOCKED"
	AlreadyLoggedIn   = "ALREADY_LOGGED_IN"
	IncorrectPassword = "INCORRECT_PASSWORD"
	TooManyAttempts   = "TOO_MANY_ATTEMPTS"
	InvalidPin        = "INVALID_PIN"
	InvalidPic        = "INVALID_PIC"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountLoggedIn = errors.New("account is currently logged in")
)

type Processor interface {
	GetOrCreate(mb *message.Buffer) func(name string, password string, automaticRegister bool) (Model, error)
	CreateAndEmit(name string, password string) (Model, error)
	Create(mb *message.Buffer) func(name string) func(password string) (Model, error)
	Update(accountId uint32, input Model) (Model, error)
	Delete(mb *message.Buffer) func(accountId uint32) error
	DeleteAndEmit(accountId uint32) error
	Login(mb *message.Buffer) func(sessionId uuid.UUID) func(accountId uint32) func(issuer string) error
	LogoutAndEmit(sessionId uuid.UUID, accountId uint32, issuer string) error
	Logout(mb *message.Buffer) func(sessionId uuid.UUID) func(accountId uint32) func(issuer string) error
	AttemptLoginAndEmit(sessionId uuid.UUID, name string, password string, ipAddress string, hwid string) error
	AttemptLogin(mb *message.Buffer) func(sessionId uuid.UUID, name string, password string, ipAddress string, hwid string) error
	ProgressStateAndEmit(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error
	ProgressState(mb *message.Buffer) func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error
	GetById(accountId uint32) (Model, error)
	GetByName(name string) (Model, error)
	GetByTenant() ([]Model, error)
	ByIdProvider(accountId uint32) model.Provider[Model]
	ByNameProvider(name string) model.Provider[Model]
	ByTenantProvider() ([]Model, error)
	LoggedInTenantProvider() ([]Model, error)
	RecordPinAttemptAndEmit(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error)
	RecordPinAttempt(mb *message.Buffer) func(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error)
	RecordPicAttemptAndEmit(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error)
	RecordPicAttempt(mb *message.Buffer) func(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

type IdOperator func(tenant.Model, uint32) error

func (p *ProcessorImpl) GetById(accountId uint32) (Model, error) {
	return p.ByIdProvider(accountId)()
}

func (p *ProcessorImpl) ByIdProvider(accountId uint32) model.Provider[Model] {
	return model.Map(decorateState(p.t))(model.Map(Make)(entityById(p.t, accountId)(p.db)))
}

func (p *ProcessorImpl) GetByName(name string) (Model, error) {
	return p.ByNameProvider(name)()
}

func (p *ProcessorImpl) ByNameProvider(name string) model.Provider[Model] {
	return model.Map(decorateState(p.t))(model.FirstProvider(model.SliceMap(Make)(entitiesByName(p.t, name)(p.db))(model.ParallelMap()), model.Filters[Model]()))
}

func (p *ProcessorImpl) GetByTenant() ([]Model, error) {
	return p.ByTenantProvider()
}

func (p *ProcessorImpl) ByTenantProvider() ([]Model, error) {
	return model.SliceMap(decorateState(p.t))(model.SliceMap(Make)(allInTenant(p.t)(p.db))(model.ParallelMap()))(model.ParallelMap())()
}

func (p *ProcessorImpl) LoggedInTenantProvider() ([]Model, error) {
	return model.FilteredProvider(p.ByTenantProvider, model.Filters[Model](LoggedIn))()
}

// allTenants Retrieves all tenants with accounts associated.
var allTenants model.Provider[[]tenant.Model] = func() ([]tenant.Model, error) {
	return GetRegistry().Tenants(context.Background()), nil
}

func decorateState(t tenant.Model) model.Transformer[Model, Model] {
	return func(m Model) (Model, error) {
		ctx := tenant.WithContext(context.Background(), t)
		st := GetRegistry().MaximalState(ctx, AccountKey{Tenant: t, AccountId: m.Id()})
		m.state = st
		return m, nil
	}
}

func GetInTransition(timeout time.Duration) ([]AccountKey, error) {
	return model.FixedProvider(GetRegistry().GetExpiredInTransition(context.Background(), timeout))()
}

func (p *ProcessorImpl) GetOrCreate(mb *message.Buffer) func(name string, password string, automaticRegister bool) (Model, error) {
	return func(name string, password string, automaticRegister bool) (Model, error) {
		m, err := p.GetByName(name)
		if err == nil {
			return m, nil
		}

		if !automaticRegister {
			p.l.Errorf("Unable to locate account by name [%s], and automatic account creation is not enabled.", name)
			return Model{}, errors.New("account not found")
		}
		return p.Create(mb)(name)(password)
	}
}

func (p *ProcessorImpl) CreateAndEmit(name string, password string) (Model, error) {
	return message.EmitWithResult[Model, string](p.p)(model.Flip(p.Create)(name))(password)
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(name string) func(password string) (Model, error) {
	return func(name string) func(password string) (Model, error) {
		return func(password string) (Model, error) {
			p.l.Debugf("Attempting to create account [%s].", name)
			hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				p.l.WithError(err).Errorf("Error generating hash when creating account [%s].", name)
				return Model{}, err
			}

			gender := byte(0)
			if p.t.Region() == "GMS" && p.t.MajorVersion() > 83 {
				gender = byte(10)
			}
			p.l.Debugf("Defaulting gender to [%d]. 0 = Male, 1 = Female, 10 = UI Choose. This is determined by Region and Version capabilities.", gender)

			m, err := create(p.db)(p.t, name, string(hashPass), gender)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create account [%s].", name)
				return Model{}, err
			}
			p.l.Debugf("Created account [%d] for [%s].", m.Id(), m.Name())
			_ = mb.Put(account2.EnvEventTopicStatus, createdEventProvider()(m.Id(), name))
			return m, nil
		}
	}
}

func (p *ProcessorImpl) Update(accountId uint32, input Model) (Model, error) {
	a, err := p.GetById(accountId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate account being updated.")
		return Model{}, err
	}

	var modifiers = make([]EntityUpdateFunction, 0)

	if a.pin != input.pin && input.pin != "" {
		p.l.Debugf("Updating PIN of account [%d].", accountId)
		modifiers = append(modifiers, updatePin(input.pin))
	}
	if a.pic != input.pic && input.pic != "" {
		p.l.Debugf("Updating PIC of account [%d].", accountId)
		modifiers = append(modifiers, updatePic(input.pic))
	}
	if a.tos != input.tos && input.tos != false {
		p.l.Debugf("Updating TOS [%t] of account [%d].", input.tos, accountId)
		modifiers = append(modifiers, updateTos(input.tos))
	}
	if a.pinAttempts != input.pinAttempts {
		p.l.Debugf("Updating PinAttempts [%d] of account [%d].", input.pinAttempts, accountId)
		modifiers = append(modifiers, updatePinAttempts(input.pinAttempts))
	}
	if a.picAttempts != input.picAttempts {
		p.l.Debugf("Updating PicAttempts [%d] of account [%d].", input.picAttempts, accountId)
		modifiers = append(modifiers, updatePicAttempts(input.picAttempts))
	}
	if a.gender != input.gender {
		p.l.Debugf("Updating Gender [%d] of account [%d].", input.gender, accountId)
		modifiers = append(modifiers, updateGender(input.gender))
	}

	if len(modifiers) == 0 {
		return a, nil
	}

	err = update(p.db)(modifiers...)(p.t, accountId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to update account.")
		return Model{}, err
	}

	return p.GetById(accountId)
}

func (p *ProcessorImpl) DeleteAndEmit(accountId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Delete(buf)(accountId)
	})
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		a, err := p.GetById(accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to locate account [%d] being deleted.", accountId)
			return ErrAccountNotFound
		}

		if a.State() != StateNotLoggedIn {
			p.l.Warnf("Cannot delete account [%d] because it is currently logged in.", accountId)
			return ErrAccountLoggedIn
		}

		err = deleteById(p.db)(p.t, accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to delete account [%d].", accountId)
			return err
		}

		GetRegistry().Terminate(p.ctx, AccountKey{Tenant: p.t, AccountId: accountId})

		p.l.Infof("Deleted account [%d] [%s].", a.Id(), a.Name())
		return mb.Put(account2.EnvEventTopicStatus, deletedEventProvider()(a.Id(), a.Name()))
	}
}

func (p *ProcessorImpl) Login(mb *message.Buffer) func(sessionId uuid.UUID) func(accountId uint32) func(issuer string) error {
	return func(sessionId uuid.UUID) func(accountId uint32) func(issuer string) error {
		return func(accountId uint32) func(issuer string) error {
			return func(issuer string) error {
				a, err := p.GetById(accountId)
				if err != nil {
					return err
				}

				ak := AccountKey{Tenant: p.t, AccountId: accountId}
				sk := ServiceKey{SessionId: sessionId, Service: Service(issuer)}
				err = GetRegistry().Login(p.ctx, ak, sk)
				if err != nil {
					return err
				}
				p.l.Debugf("State transition triggered a login.")
				return mb.Put(account2.EnvEventTopicStatus, loggedInEventProvider()(a.Id(), a.Name()))
			}
		}
	}
}

func (p *ProcessorImpl) LogoutAndEmit(sessionId uuid.UUID, accountId uint32, issuer string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Logout(buf)(sessionId)(accountId)(issuer)
	})
}

func (p *ProcessorImpl) Logout(mb *message.Buffer) func(sessionId uuid.UUID) func(accountId uint32) func(issuer string) error {
	return func(sessionId uuid.UUID) func(accountId uint32) func(issuer string) error {
		return func(accountId uint32) func(issuer string) error {
			return func(issuer string) error {
				a, err := p.GetById(accountId)
				if err != nil {
					return err
				}

				if sessionId == uuid.Nil {
					ok := GetRegistry().Terminate(p.ctx, AccountKey{Tenant: p.t, AccountId: accountId})
					if !ok {
						return errors.New("error while logging out")
					}
				} else {
					ok := GetRegistry().Logout(p.ctx, AccountKey{Tenant: p.t, AccountId: accountId}, ServiceKey{SessionId: sessionId, Service: Service(issuer)})
					if !ok {
						return errors.New("error while logging out")
					}
				}
				p.l.Debugf("Logging out [%d] for [%s] via session [%s].", accountId, issuer, sessionId.String())
				return mb.Put(account2.EnvEventTopicStatus, loggedOutEventProvider()(a.Id(), a.Name()))
			}
		}
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
			err := model.ForEachSlice(NewProcessor(l, ctx, db).LoggedInTenantProvider, func(a Model) error {
				l.Debugf("Logging out [%d] [%s]. Triggered by [%s].", a.Id(), a.Name(), "service shutdown")
				return producer.ProviderImpl(l)(ctx)(account2.EnvEventTopicStatus)(loggedOutEventProvider()(a.Id(), a.Name()))
			}, model.ParallelExecute())
			if err != nil {
				l.WithError(err).Errorf("Error destroying all monsters on teardown.")
			}
			return nil
		}
	}
}

func (p *ProcessorImpl) AttemptLoginAndEmit(sessionId uuid.UUID, name string, password string, ipAddress string, hwid string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.AttemptLogin(buf)(sessionId, name, password, ipAddress, hwid)
	})
}

func (p *ProcessorImpl) AttemptLogin(mb *message.Buffer) func(sessionId uuid.UUID, name string, password string, ipAddress string, hwid string) error {
	return func(sessionId uuid.UUID, name string, password string, ipAddress string, hwid string) error {
		p.l.Debugf("Attemting login for [%s].", name)
		if checkLoginAttempts(sessionId) > 4 {
			p.l.Warnf("Session [%s] has attempted to log into (or create) an account too many times.", sessionId.String())
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, 0, "", TooManyAttempts, ipAddress, hwid))
		}

		c, err := configuration.Get()
		if err != nil {
			p.l.WithError(err).Errorf("Error reading needed configuration.")
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, 0, "", SystemError, ipAddress, hwid))
		}

		a, err := p.GetOrCreate(mb)(name, password, c.AutomaticRegister)
		if err != nil && !c.AutomaticRegister {
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, 0, "", NotRegistered, ipAddress, hwid))
		}
		if err != nil {
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, 0, "", SystemError, ipAddress, hwid))
		}

		checkResult, err := ban.CheckBan(p.l, p.ctx, ipAddress, hwid, a.Id())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to check ban status for account [%d]. Proceeding with fail-open strategy.", a.Id())
		} else if checkResult.Banned {
			p.l.Infof("Account [%d] is banned. type=[%d] reason=[%s].", a.Id(), checkResult.BanType, checkResult.Reason)
			return mb.Put(account2.EnvEventSessionStatusTopic, banStatusProvider(sessionId, a.Id(), a.Name(), ipAddress, hwid, checkResult.ReasonCode, checkResult.ExpiresAt))
		}

		if a.State() != StateNotLoggedIn {
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), a.Name(), AlreadyLoggedIn, ipAddress, hwid))
		}
		if a.Password()[0] != uint8('$') || a.Password()[1] != uint8('2') || bcrypt.CompareHashAndPassword([]byte(a.Password()), []byte(password)) != nil {
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), a.Name(), IncorrectPassword, ipAddress, hwid))
		}

		err = p.Login(mb)(sessionId)(a.Id())(ServiceLogin)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to record login.")
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), a.Name(), SystemError, ipAddress, hwid))
		}

		p.l.Debugf("Login successful for [%s].", name)

		if !a.TOS() && p.t.Region() != "JMS" {
			return mb.Put(account2.EnvEventSessionStatusTopic, requestLicenseAgreementStatusProvider(sessionId, a.Id(), a.Name()))
		}
		return mb.Put(account2.EnvEventSessionStatusTopic, createdStatusProvider(sessionId, a.Id(), a.Name(), ipAddress, hwid))
	}
}

func (p *ProcessorImpl) ProgressStateAndEmit(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.ProgressState(buf)(sessionId, issuer, accountId, state, params)
	})
}

func (p *ProcessorImpl) ProgressState(mb *message.Buffer) func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
	return func(sessionId uuid.UUID, issuer string, accountId uint32, state State, params interface{}) error {
		a, err := p.GetById(accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to locate account a session is being created for.")
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), "", NotRegistered, "", ""))
		}

		p.l.Debugf("Received request to progress state for account [%d] to state [%d] from state [%d].", accountId, state, a.State())
		for k, v := range GetRegistry().GetStates(p.ctx, AccountKey{Tenant: p.t, AccountId: accountId}) {
			p.l.Debugf("Has state [%d] for [%s] via session [%s].", v.State, k.Service, k.SessionId.String())
		}
		if a.State() == StateNotLoggedIn {
			return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), a.Name(), SystemError, "", ""))
		}
		if state == StateNotLoggedIn {
			err = p.Logout(mb)(sessionId)(accountId)(issuer)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to logout account.")
				return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), a.Name(), SystemError, "", ""))
			}
			return mb.Put(account2.EnvEventSessionStatusTopic, stateChangedStatusProvider(sessionId, a.Id(), a.Name(), uint8(StateNotLoggedIn), params))
		}
		if state == StateLoggedIn {
			err = p.Login(mb)(sessionId)(accountId)(issuer)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to login account.")
				return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, a.Id(), a.Name(), SystemError, "", ""))
			}
			return mb.Put(account2.EnvEventSessionStatusTopic, stateChangedStatusProvider(sessionId, a.Id(), a.Name(), uint8(StateLoggedIn), params))
		}
		if state == StateTransition {
			err = GetRegistry().Transition(p.ctx, AccountKey{Tenant: p.t, AccountId: accountId}, ServiceKey{SessionId: sessionId, Service: Service(issuer)})
			if err == nil {
				p.l.Debugf("State transition triggered a transition.")
			}
			return mb.Put(account2.EnvEventSessionStatusTopic, stateChangedStatusProvider(sessionId, a.Id(), a.Name(), uint8(StateTransition), params))
		}
		return mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(sessionId, 0, "", SystemError, "", ""))
	}
}

func (p *ProcessorImpl) RecordPinAttemptAndEmit(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
	var attempts int
	var limitReached bool
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var innerErr error
		attempts, limitReached, innerErr = p.RecordPinAttempt(buf)(accountId, success, ipAddress, hwid)
		return innerErr
	})
	return attempts, limitReached, err
}

func (p *ProcessorImpl) RecordPinAttempt(mb *message.Buffer) func(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
	return func(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
		a, err := p.GetById(accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to locate account [%d] for PIN attempt recording.", accountId)
			return 0, false, err
		}

		if success {
			if a.PinAttempts() > 0 {
				p.l.Debugf("Resetting PIN attempts for account [%d] after successful PIN entry.", accountId)
				err = update(p.db)(updatePinAttempts(0))(p.t, accountId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to reset PIN attempts for account [%d].", accountId)
					return 0, false, err
				}
			}
			return 0, false, nil
		}

		newAttempts := a.PinAttempts() + 1
		p.l.Debugf("Recording failed PIN attempt [%d] for account [%d].", newAttempts, accountId)

		_ = mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(uuid.Nil, accountId, a.Name(), InvalidPin, ipAddress, hwid))

		c, err := configuration.Get()
		if err != nil {
			p.l.WithError(err).Errorf("Unable to read configuration for PIN attempt limit.")
			return newAttempts, false, err
		}

		limitReached := newAttempts >= c.MaxPinAttempts
		if limitReached {
			p.l.Warnf("Account [%d] has reached the PIN attempt limit [%d]. Issuing temporary ban.", accountId, c.MaxPinAttempts)

			duration, err := time.ParseDuration(c.PinBanDuration)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to parse PIN ban duration [%s]. Defaulting to 15 minutes.", c.PinBanDuration)
				duration = 15 * time.Minute
			}
			expiresAt := time.Now().Add(duration)
			reason := fmt.Sprintf("Exceeded maximum PIN attempts (%d)", c.MaxPinAttempts)

			err = mb.Put(ban2.EnvCommandTopic, createBanCommandProvider(accountId, reason, ReasonCodeInvalidAttempts, expiresAt))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to emit ban command for account [%d].", accountId)
				return newAttempts, true, err
			}

			// Reset counter after issuing ban
			err = update(p.db)(updatePinAttempts(0))(p.t, accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to reset PIN attempts after ban for account [%d].", accountId)
				return 0, true, err
			}
			return 0, true, nil
		}

		err = update(p.db)(updatePinAttempts(newAttempts))(p.t, accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update PIN attempts for account [%d].", accountId)
			return newAttempts, false, err
		}
		return newAttempts, false, nil
	}
}

func (p *ProcessorImpl) RecordPicAttemptAndEmit(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
	var attempts int
	var limitReached bool
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var innerErr error
		attempts, limitReached, innerErr = p.RecordPicAttempt(buf)(accountId, success, ipAddress, hwid)
		return innerErr
	})
	return attempts, limitReached, err
}

func (p *ProcessorImpl) RecordPicAttempt(mb *message.Buffer) func(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
	return func(accountId uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
		a, err := p.GetById(accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to locate account [%d] for PIC attempt recording.", accountId)
			return 0, false, err
		}

		if success {
			if a.PicAttempts() > 0 {
				p.l.Debugf("Resetting PIC attempts for account [%d] after successful PIC entry.", accountId)
				err = update(p.db)(updatePicAttempts(0))(p.t, accountId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to reset PIC attempts for account [%d].", accountId)
					return 0, false, err
				}
			}
			return 0, false, nil
		}

		newAttempts := a.PicAttempts() + 1
		p.l.Debugf("Recording failed PIC attempt [%d] for account [%d].", newAttempts, accountId)

		_ = mb.Put(account2.EnvEventSessionStatusTopic, errorStatusProvider(uuid.Nil, accountId, a.Name(), InvalidPic, ipAddress, hwid))

		c, err := configuration.Get()
		if err != nil {
			p.l.WithError(err).Errorf("Unable to read configuration for PIC attempt limit.")
			return newAttempts, false, err
		}

		limitReached := newAttempts >= c.MaxPicAttempts
		if limitReached {
			p.l.Warnf("Account [%d] has reached the PIC attempt limit [%d]. Issuing temporary ban.", accountId, c.MaxPicAttempts)

			duration, err := time.ParseDuration(c.PicBanDuration)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to parse PIC ban duration [%s]. Defaulting to 15 minutes.", c.PicBanDuration)
				duration = 15 * time.Minute
			}
			expiresAt := time.Now().Add(duration)
			reason := fmt.Sprintf("Exceeded maximum PIC attempts (%d)", c.MaxPicAttempts)

			err = mb.Put(ban2.EnvCommandTopic, createBanCommandProvider(accountId, reason, ReasonCodeInvalidAttempts, expiresAt))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to emit ban command for account [%d].", accountId)
				return newAttempts, true, err
			}

			// Reset counter after issuing ban
			err = update(p.db)(updatePicAttempts(0))(p.t, accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to reset PIC attempts after ban for account [%d].", accountId)
				return 0, true, err
			}
			return 0, true, nil
		}

		err = update(p.db)(updatePicAttempts(newAttempts))(p.t, accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update PIC attempts for account [%d].", accountId)
			return newAttempts, false, err
		}
		return newAttempts, false, nil
	}
}

func checkLoginAttempts(sessionId uuid.UUID) byte {
	return 0
}
