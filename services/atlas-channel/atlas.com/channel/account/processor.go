package account

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type LoginErr string

// Processor interface defines the operations for account processing
type Processor interface {
	ByIdModelProvider(id uint32) model.Provider[Model]
	AllProvider() model.Provider[[]Model]
	GetById(id uint32) (Model, error)
	GetAllAccounts() ([]Model, error)
	IsLoggedIn(id uint32) bool
	InitializeRegistry() error
	RecordPicAttempt(id uint32, success bool, ipAddress string, hwid string) (int, bool, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountById(p.ctx, id), Extract)
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	root, err := getBaseRequest(p.ctx)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(root+AccountsResource, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdModelProvider(id)()
}

func (p *ProcessorImpl) GetAllAccounts() ([]Model, error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) IsLoggedIn(id uint32) bool {
	return GetRegistry().LoggedIn(Key{Tenant: tenant.MustFromContext(p.ctx), Id: id})
}

func (p *ProcessorImpl) InitializeRegistry() error {
	as, err := model.CollectToMap[Model, Key, bool](p.AllProvider(), KeyForTenantFunc(tenant.MustFromContext(p.ctx)), IsLogged)()
	if err != nil {
		return err
	}
	GetRegistry().Init(as)
	return nil
}

func IsLogged(m Model) bool {
	return m.LoggedIn() > 0
}

// RecordPicAttempt records a PIC-comparison outcome against
// accounts/{accountId}/pic-attempts and returns the running attempt count and
// whether the lockout limit was reached. It is the lockout counter behind the
// credential validated by the cash-shop NAME_TRANSFER / WORLD_TRANSFER check
// handlers (task-227 Task 26 ruling 4) — without it, those check ops are a
// brute-force oracle against the account's second password / birthday code.
func (p *ProcessorImpl) RecordPicAttempt(id uint32, success bool, ipAddress string, hwid string) (int, bool, error) {
	result, err := requestRecordPicAttempt(p.ctx, id, success, ipAddress, hwid)(p.l, p.ctx)
	if err != nil {
		return 0, false, err
	}
	return result.Attempts, result.LimitReached, nil
}
