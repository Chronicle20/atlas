package character

import (
	"atlas-login/inventory"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"regexp"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	ip  *inventory.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		ip:  inventory.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) IsValidName(name string) (bool, error) {
	m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
	if err != nil {
		return false, err
	}
	if !m {
		return false, nil
	}

	cs, err := p.GetByName()(name)
	if len(cs) != 0 || err != nil {
		return false, nil
	}

	//TODO
	//bn, err := blocked_name.IsBlockedName(l, span)(name)
	//if bn {
	//	return false, err
	//}

	return true, nil
}

func (p *Processor) ByAccountAndWorldProvider(decorators ...model.Decorator[Model]) func(accountId uint32, worldId byte) model.Provider[[]Model] {
	return func(accountId uint32, worldId byte) model.Provider[[]Model] {
		mp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByAccountAndWorld(accountId, worldId), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())
	}
}

func (p *Processor) GetForWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId byte) ([]Model, error) {
	return func(accountId uint32, worldId byte) ([]Model, error) {
		cs, err := p.ByAccountAndWorldProvider(decorators...)(accountId, worldId)()
		if errors.Is(requests.ErrNotFound, err) {
			return make([]Model, 0), nil
		}
		return cs, err
	}
}

func (p *Processor) ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		mp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())
	}
}

func (p *Processor) GetByName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		return p.ByNameProvider(decorators...)(name)()
	}
}

func (p *Processor) ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		mp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(id), Extract)
		return model.Map(model.Decorate(decorators))(mp)
	}
}

func (p *Processor) GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return p.ByIdProvider(decorators...)(id)()
	}
}

func (p *Processor) InventoryDecorator() model.Decorator[Model] {
	return func(m Model) Model {
		i, err := p.ip.GetByCharacterId(m.Id())
		if err != nil {
			return m
		}
		return m.SetInventory(i)
	}
}

func (p *Processor) DeleteById(characterId uint32) error {
	return requestDelete(characterId)(p.l, p.ctx)
}
