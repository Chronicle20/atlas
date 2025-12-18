package character

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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

func (p *Processor) GetMap(characterId uint32) (_map.Model, error) {
	mk, ok := getRegistry().GetMap(characterId)
	if !ok {
		return _map.Model{}, errors.New("not found")
	}
	m := _map.NewModel(world.Id(mk.WorldId))(channel.Id(mk.ChannelId))(_map.Id(mk.MapId))
	return m, nil
}

func (p *Processor) Enter(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	getRegistry().AddCharacter(MapKey{Tenant: p.t, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
}

func (p *Processor) Exit(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	getRegistry().RemoveCharacter(characterId)
}

func (p *Processor) TransitionMap(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	p.Enter(worldId, channelId, mapId, characterId)
}

func (p *Processor) TransitionChannel(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
	p.Enter(worldId, channelId, mapId, characterId)
}
