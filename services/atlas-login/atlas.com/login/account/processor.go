package account

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type LoginErr string

type Processor interface {
	ForAccountByName(name string, operator model.Operator[Model])
	ForAccountById(id uint32, operator model.Operator[Model])
	ByNameModelProvider(name string) model.Provider[Model]
	ByIdModelProvider(id uint32) model.Provider[Model]
	AllProvider() model.Provider[[]Model]
	GetById(id uint32) (Model, error)
	GetByName(name string) (Model, error)
	IsLoggedIn(id uint32) bool
	InitializeRegistry() error
	UpdatePin(id uint32, pin string) error
	UpdatePic(id uint32, pic string) error
	UpdateTos(id uint32, tos bool) error
	UpdateGender(id uint32, gender byte) error
}

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

func (p *ProcessorImpl) ForAccountByName(name string, operator model.Operator[Model]) {
	_ = model.For[Model](p.ByNameModelProvider(name), operator)
}

func (p *ProcessorImpl) ForAccountById(id uint32, operator model.Operator[Model]) {
	_ = model.For[Model](p.ByIdModelProvider(id), operator)
}

func (p *ProcessorImpl) ByNameModelProvider(name string) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountByName(name), Extract)
}

func (p *ProcessorImpl) ByIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountById(id), Extract)
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAccounts(), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdModelProvider(id)()
}

func (p *ProcessorImpl) GetByName(name string) (Model, error) {
	return p.ByNameModelProvider(name)()
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

func (p *ProcessorImpl) UpdatePin(id uint32, pin string) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	updated := a.ToBuilder().SetPin(pin).Build()
	_, err = requestUpdate(updated)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProcessorImpl) UpdatePic(id uint32, pic string) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	updated := a.ToBuilder().SetPic(pic).Build()
	_, err = requestUpdate(updated)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProcessorImpl) UpdateTos(id uint32, tos bool) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	updated := a.ToBuilder().SetTos(tos).Build()
	_, err = requestUpdate(updated)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProcessorImpl) UpdateGender(id uint32, gender byte) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	updated := a.ToBuilder().SetGender(gender).Build()
	_, err = requestUpdate(updated)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}
