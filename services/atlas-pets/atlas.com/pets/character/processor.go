package character

import (
	"atlas-pets/inventory"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)
	InventoryDecorator(m Model) Model
	Enter(worldId byte, channelId byte, mapId uint32, characterId uint32)
	Exit(worldId byte, channelId byte, mapId uint32, characterId uint32)
	TransitionMap(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32)
	TransitionChannel(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	ip  inventory.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
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

func (p *ProcessorImpl) InventoryDecorator(m Model) Model {
	i, err := p.ip.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetInventory(i)
}

func GetLoggedIn() model.Provider[map[uint32]MapKey] {
	return model.FixedProvider(getRegistry().GetLoggedIn())
}

func (p *ProcessorImpl) Enter(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	getRegistry().AddCharacter(characterId, MapKey{Tenant: p.t, WorldId: worldId, ChannelId: channelId, MapId: mapId})
}

func (p *ProcessorImpl) Exit(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	getRegistry().RemoveCharacter(characterId)
}

func (p *ProcessorImpl) TransitionMap(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	p.Exit(worldId, channelId, oldMapId, characterId)
	p.Enter(worldId, channelId, mapId, characterId)
}

func (p *ProcessorImpl) TransitionChannel(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
	p.Exit(worldId, oldChannelId, mapId, characterId)
	p.Enter(worldId, channelId, mapId, characterId)
}
