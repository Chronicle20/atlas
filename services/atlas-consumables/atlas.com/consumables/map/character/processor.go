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
	mk, ok := getRegistry().GetMap(characterId)
	if !ok {
		return field.Model{}, errors.New("not found")
	}
	return mk.Field, nil
}

func (p *Processor) Enter(f field.Model, characterId uint32) {
	getRegistry().AddCharacter(MapKey{Tenant: p.t, Field: f}, characterId)
}

func (p *Processor) Exit(_ field.Model, characterId uint32) {
	getRegistry().RemoveCharacter(characterId)
}

func (p *Processor) TransitionMap(f field.Model, characterId uint32) {
	p.Enter(f, characterId)
}

func (p *Processor) TransitionChannel(f field.Model, characterId uint32) {
	p.Enter(f, characterId)
}
