package account

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type LoginErr string

// Processor interface defines the operations for account processing
type Processor interface {
	ByIdModelProvider(id uint32) model.Provider[Model]
	AllProvider() model.Provider[[]Model]
	GetById(id uint32) (Model, error)
	GetAll() ([]Model, error)
	IsLoggedIn(id uint32) bool
	InitializeRegistry() error
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

func (p *ProcessorImpl) ByIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountById(id), Extract)
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAccounts, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdModelProvider(id)()
}

func (p *ProcessorImpl) GetAll() ([]Model, error) {
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
