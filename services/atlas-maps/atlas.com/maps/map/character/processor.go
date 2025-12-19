package character

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
	GetMapsWithCharacters() []MapKey
	Enter(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
	Exit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error) {
	t := tenant.MustFromContext(p.ctx)
	return getRegistry().GetInMap(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}), nil
}

func (p *ProcessorImpl) GetMapsWithCharacters() []MapKey {
	return getRegistry().GetMapsWithCharacters()
}

func (p *ProcessorImpl) Enter(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRegistry().AddCharacter(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
}

func (p *ProcessorImpl) Exit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRegistry().RemoveCharacter(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
}
