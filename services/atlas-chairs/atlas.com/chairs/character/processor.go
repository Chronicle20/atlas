package character

import (
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapProvider(field field.Model) model.Provider[[]uint32]
	GetCharactersInMap(field field.Model) ([]uint32, error)
	Enter(field field.Model, characterId uint32)
	Exit(field field.Model, characterId uint32)
	TransitionMap(oldField field.Model, newField field.Model, characterId uint32)
	TransitionChannel(oldField field.Model, newField field.Model, characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) InMapProvider(field field.Model) model.Provider[[]uint32] {
	cids := getRegistry().GetInMap(MapKey{Tenant: p.t, WorldId: field.WorldId(), ChannelId: field.ChannelId(), MapId: field.MapId()})
	return model.FixedProvider(cids)
}

func (p *ProcessorImpl) GetCharactersInMap(field field.Model) ([]uint32, error) {
	return p.InMapProvider(field)()
}

func (p *ProcessorImpl) Enter(field field.Model, characterId uint32) {
	getRegistry().AddCharacter(MapKey{Tenant: p.t, WorldId: field.WorldId(), ChannelId: field.ChannelId(), MapId: field.MapId()}, characterId)
}

func (p *ProcessorImpl) Exit(field field.Model, characterId uint32) {
	getRegistry().RemoveCharacter(MapKey{Tenant: p.t, WorldId: field.WorldId(), ChannelId: field.ChannelId(), MapId: field.MapId()}, characterId)
}

func (p *ProcessorImpl) TransitionMap(oldField field.Model, newField field.Model, characterId uint32) {
	p.Exit(oldField, characterId)
	p.Enter(newField, characterId)
}

func (p *ProcessorImpl) TransitionChannel(oldField field.Model, newField field.Model, characterId uint32) {
	p.Exit(oldField, characterId)
	p.Enter(newField, characterId)
}
