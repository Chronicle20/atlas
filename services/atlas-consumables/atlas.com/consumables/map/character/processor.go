package character

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetMap(characterId uint32) (field.Model, error)
	Enter(f field.Model, characterId uint32)
	Exit(_ field.Model, characterId uint32)
	TransitionMap(f field.Model, characterId uint32)
	TransitionChannel(f field.Model, characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetMap(characterId uint32) (field.Model, error) {
	f, ok := GetRegistry().GetMap(p.ctx, characterId)
	if !ok {
		return field.Model{}, errors.New("not found")
	}
	return f, nil
}

func (p *ProcessorImpl) Enter(f field.Model, characterId uint32) {
	GetRegistry().AddCharacter(p.ctx, characterId, f)
}

func (p *ProcessorImpl) Exit(_ field.Model, characterId uint32) {
	GetRegistry().RemoveCharacter(p.ctx, characterId)
}

func (p *ProcessorImpl) TransitionMap(f field.Model, characterId uint32) {
	p.Enter(f, characterId)
}

func (p *ProcessorImpl) TransitionChannel(f field.Model, characterId uint32) {
	p.Enter(f, characterId)
}
