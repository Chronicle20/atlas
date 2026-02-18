package character

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/field"
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

func (p *Processor) GetMap(characterId uint32) (field.Model, error) {
	f, ok := GetRegistry().GetMap(p.ctx, characterId)
	if !ok {
		return field.Model{}, errors.New("not found")
	}
	return f, nil
}

func (p *Processor) Enter(f field.Model, characterId uint32) {
	GetRegistry().AddCharacter(p.ctx, characterId, f)
}

func (p *Processor) Exit(_ field.Model, characterId uint32) {
	GetRegistry().RemoveCharacter(p.ctx, characterId)
}

func (p *Processor) TransitionMap(f field.Model, characterId uint32) {
	p.Enter(f, characterId)
}

func (p *Processor) TransitionChannel(f field.Model, characterId uint32) {
	p.Enter(f, characterId)
}
