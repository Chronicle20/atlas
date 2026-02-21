package character

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error)
	GetCharactersInMapAllInstances(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
	GetMapsWithCharacters() []MapKey
	Enter(transactionId uuid.UUID, f field.Model, characterId uint32)
	Exit(transactionId uuid.UUID, f field.Model, characterId uint32)
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

func (p *ProcessorImpl) GetCharactersInMap(_ uuid.UUID, f field.Model) ([]uint32, error) {
	t := tenant.MustFromContext(p.ctx)
	return getRegistry().GetInMap(MapKey{Tenant: t, Field: f}), nil
}

func (p *ProcessorImpl) GetCharactersInMapAllInstances(_ uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error) {
	t := tenant.MustFromContext(p.ctx)
	return getRegistry().GetInMapAllInstances(t, worldId, channelId, mapId), nil
}

func (p *ProcessorImpl) GetMapsWithCharacters() []MapKey {
	return getRegistry().GetMapsWithCharacters()
}

func (p *ProcessorImpl) Enter(_ uuid.UUID, f field.Model, characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRegistry().AddCharacter(MapKey{Tenant: t, Field: f}, characterId)
}

func (p *ProcessorImpl) Exit(_ uuid.UUID, f field.Model, characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRegistry().RemoveCharacter(MapKey{Tenant: t, Field: f}, characterId)
}
