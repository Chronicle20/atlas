package character

import (
	"atlas-npc/inventory"
	character2 "atlas-npc/kafka/message/character"
	"atlas-npc/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)
	ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model]
	GetByName(decorators ...model.Decorator[Model]) func(name string) (Model, error)
	IdByNameProvider(name string) model.Provider[uint32]
	InventoryDecorator(m Model) Model
	RequestChangeMeso(worldId world.Id, characterId uint32, actorId uint32, actorType string, amount int32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	ip  inventory.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		ip:  inventory.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		cp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
		return model.Map(model.Decorate(decorators))(cp)()
	}
}

func (p *ProcessorImpl) ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		ps := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(ps)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetByName(decorators ...model.Decorator[Model]) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		return model.First(p.ByNameProvider(decorators...)(name), model.Filters[Model]())
	}
}

func (p *ProcessorImpl) IdByNameProvider(name string) model.Provider[uint32] {
	c, err := p.GetByName()(name)
	if err != nil {
		return model.ErrorProvider[uint32](err)
	}
	return model.FixedProvider(c.Id())
}

func (p *ProcessorImpl) InventoryDecorator(m Model) Model {
	i, err := p.ip.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetInventory(i)
}

func (p *ProcessorImpl) RequestChangeMeso(worldId world.Id, characterId uint32, actorId uint32, actorType string, amount int32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(RequestChangeMesoCommandProvider(characterId, worldId, actorId, actorType, amount))
}
