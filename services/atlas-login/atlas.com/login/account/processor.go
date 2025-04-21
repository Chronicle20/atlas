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

func (p *Processor) ForAccountByName(name string, operator model.Operator[Model]) {
	_ = model.For[Model](p.ByNameModelProvider(name), operator)
}

func (p *Processor) ForAccountById(id uint32, operator model.Operator[Model]) {
	_ = model.For[Model](p.ByIdModelProvider(id), operator)
}

func (p *Processor) ByNameModelProvider(name string) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountByName(name), Extract)
}

func (p *Processor) ByIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestAccountById(id), Extract)
}

func (p *Processor) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAccounts(), Extract, model.Filters[Model]())
}

func (p *Processor) GetById(id uint32) (Model, error) {
	return p.ByIdModelProvider(id)()
}

func (p *Processor) GetByName(name string) (Model, error) {
	return p.ByNameModelProvider(name)()
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

func (p *Processor) UpdatePin(id uint32, pin string) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	a.pin = pin
	_, err = requestUpdate(a)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *Processor) UpdatePic(id uint32, pic string) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	a.pic = pic
	_, err = requestUpdate(a)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *Processor) UpdateTos(id uint32, tos bool) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	a.tos = tos
	_, err = requestUpdate(a)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *Processor) UpdateGender(id uint32, gender byte) error {
	a, err := p.GetById(id)
	if err != nil {
		return err
	}
	a.gender = gender
	_, err = requestUpdate(a)(p.l, p.ctx)
	if err != nil {
		return err
	}
	return nil
}
