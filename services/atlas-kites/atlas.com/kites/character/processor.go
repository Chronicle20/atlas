package character

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) InMapProvider(f field.Model) model.Provider[[]uint32] {
	return func() ([]uint32, error) {
		return getRegistry().GetInMap(p.ctx, MapKey{Tenant: p.t, Field: f})
	}
}

func (p *ProcessorImpl) GetCharactersInMap(f field.Model) ([]uint32, error) {
	return p.InMapProvider(f)()
}

func (p *ProcessorImpl) Enter(f field.Model, characterId uint32) {
	if err := getRegistry().AddCharacter(p.ctx, MapKey{Tenant: p.t, Field: f}, characterId); err != nil {
		p.l.WithError(err).Errorf("Unable to add character [%d] to field character index.", characterId)
	}
}

func (p *ProcessorImpl) Exit(f field.Model, characterId uint32) {
	if err := getRegistry().RemoveCharacter(p.ctx, MapKey{Tenant: p.t, Field: f}, characterId); err != nil {
		p.l.WithError(err).Errorf("Unable to remove character [%d] from field character index.", characterId)
	}
}

func (p *ProcessorImpl) TransitionMap(of field.Model, nf field.Model, characterId uint32) {
	p.Exit(of, characterId)
	p.Enter(nf, characterId)
}

func (p *ProcessorImpl) TransitionChannel(of field.Model, nf field.Model, characterId uint32) {
	p.Exit(of, characterId)
	p.Enter(nf, characterId)
}
