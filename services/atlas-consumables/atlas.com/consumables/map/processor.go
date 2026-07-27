package _map

import (
	"atlas-consumables/character"
	"atlas-consumables/portal"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	WarpRandom(f field.Model) func(characterId uint32) error
	WarpToPortal(f field.Model, characterId uint32, pp model.Provider[uint32]) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  character.Processor
	pp  portal.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		pp:  portal.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WarpRandom(f field.Model) func(characterId uint32) error {
	return func(characterId uint32) error {
		return p.WarpToPortal(f, characterId, p.pp.RandomSpawnPointIdProvider(f.MapId()))
	}
}

func (p *ProcessorImpl) WarpToPortal(f field.Model, characterId uint32, pp model.Provider[uint32]) error {
	id, err := pp()
	if err != nil {
		return err
	}
	return p.cp.ChangeMap(f, characterId, id)
}
