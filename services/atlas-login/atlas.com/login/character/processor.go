package character

import (
	"atlas-login/inventory"
	"context"
	"errors"
	"regexp"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	IsValidName(name string) (bool, error)
	ByAccountAndWorldProvider(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) model.Provider[[]Model]
	GetForWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error)
	ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model]
	GetByName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error)
	ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model]
	GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error)
	InventoryDecorator() model.Decorator[Model]
	DeleteById(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	ip  *inventory.ProcessorImpl
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		ip:  inventory.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) IsValidName(name string) (bool, error) {
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

func (p *ProcessorImpl) ByAccountAndWorldProvider(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) model.Provider[[]Model] {
	return func(accountId uint32, worldId world.Id) model.Provider[[]Model] {
		mp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByAccountAndWorld(accountId, worldId), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetForWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error) {
	return func(accountId uint32, worldId world.Id) ([]Model, error) {
		cs, err := p.ByAccountAndWorldProvider(decorators...)(accountId, worldId)()
		if errors.Is(err, requests.ErrNotFound) {
			return make([]Model, 0), nil
		}
		return cs, err
	}
}

func (p *ProcessorImpl) ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		mp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetByName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		return p.ByNameProvider(decorators...)(name)()
	}
}

func (p *ProcessorImpl) ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		mp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(id), Extract)
		return model.Map(model.Decorate(decorators))(mp)
	}
}

func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return p.ByIdProvider(decorators...)(id)()
	}
}

func (p *ProcessorImpl) InventoryDecorator() model.Decorator[Model] {
	return func(m Model) Model {
		i, err := p.ip.GetByCharacterId(m.Id())
		if err != nil {
			return m
		}
		return m.SetInventory(i)
	}
}

func (p *ProcessorImpl) DeleteById(characterId uint32) error {
	return requestDelete(characterId)(p.l, p.ctx)
}
