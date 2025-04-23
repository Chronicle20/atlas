package character

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

func (p *Processor) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		cp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
		return model.Map(model.Decorate(decorators))(cp)()
	}
}

func GetLoggedIn() model.Provider[map[uint32]MapKey] {
	return model.FixedProvider(getRegistry().GetLoggedIn())
}

func (p *Processor) Enter(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	getRegistry().AddCharacter(characterId, MapKey{Tenant: p.t, WorldId: worldId, ChannelId: channelId, MapId: mapId})
}

func (p *Processor) Exit(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	getRegistry().RemoveCharacter(characterId)
}

func (p *Processor) TransitionMap(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	p.Exit(worldId, channelId, oldMapId, characterId)
	p.Enter(worldId, channelId, mapId, characterId)
}

func (p *Processor) TransitionChannel(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
	p.Exit(worldId, oldChannelId, mapId, characterId)
	p.Enter(worldId, channelId, mapId, characterId)
}
