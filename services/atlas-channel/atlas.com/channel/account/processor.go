package account

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type LoginErr string

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountById(id), Extract)
}

func (p *Processor) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAccounts, Extract, model.Filters[Model]())
}

func (p *Processor) GetById(id uint32) (Model, error) {
	return p.ByIdModelProvider(id)()
}

func (p *Processor) GetAll() ([]Model, error) {
	return p.AllProvider()()
}

func (p *Processor) IsLoggedIn(id uint32) bool {
	return GetRegistry().LoggedIn(Key{Tenant: tenant.MustFromContext(p.ctx), Id: id})
}

func (p *Processor) InitializeRegistry() error {
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
