package character

import (
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapProvider(f field.Model) model.Provider[[]uint32]
	GetCharactersInMap(f field.Model) ([]uint32, error)
	Enter(f field.Model, characterId uint32)
	Exit(f field.Model, characterId uint32)
	TransitionMap(of field.Model, nf field.Model, characterId uint32)
	TransitionChannel(of field.Model, nf field.Model, characterId uint32)
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

func (p *ProcessorImpl) InMapProvider(f field.Model) model.Provider[[]uint32] {
	cids := getRegistry().GetInMap(MapKey{Tenant: p.t, WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId()})
	return model.FixedProvider(cids)
}

func (p *ProcessorImpl) GetCharactersInMap(f field.Model) ([]uint32, error) {
	return p.InMapProvider(f)()
}

func (p *ProcessorImpl) Enter(f field.Model, characterId uint32) {
	getRegistry().AddCharacter(MapKey{Tenant: p.t, WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId()}, characterId)
}

func (p *ProcessorImpl) Exit(f field.Model, characterId uint32) {
	getRegistry().RemoveCharacter(MapKey{Tenant: p.t, WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId()}, characterId)
}

func (p *ProcessorImpl) TransitionMap(of field.Model, nf field.Model, characterId uint32) {
	p.Exit(of, characterId)
	p.Enter(nf, characterId)
}

func (p *ProcessorImpl) TransitionChannel(of field.Model, nf field.Model, characterId uint32) {
	p.Exit(of, characterId)
	p.Enter(nf, characterId)
}
