package character

import (
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error)
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

func (p *ProcessorImpl) GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
	t := tenant.MustFromContext(p.ctx)
	return getRegistry().GetInMap(MapKey{Tenant: t, WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance()}), nil
}

func (p *ProcessorImpl) GetMapsWithCharacters() []MapKey {
	return getRegistry().GetMapsWithCharacters()
}

func (p *ProcessorImpl) Enter(transactionId uuid.UUID, f field.Model, characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRegistry().AddCharacter(MapKey{Tenant: t, WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance()}, characterId)
}

func (p *ProcessorImpl) Exit(transactionId uuid.UUID, f field.Model, characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRegistry().RemoveCharacter(MapKey{Tenant: t, WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance()}, characterId)
}
