package _map

import (
	"atlas-consumables/character"
	"atlas-consumables/portal"
	"context"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  *character.Processor
	pp  *portal.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		pp:  portal.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) WarpRandom(m _map2.Model) func(characterId uint32) error {
	return func(characterId uint32) error {
		return p.WarpToPortal(m, characterId, p.pp.RandomSpawnPointIdProvider(m.MapId()))
	}
}

func (p *Processor) WarpToPortal(m _map2.Model, characterId uint32, pp model.Provider[uint32]) error {
	id, err := pp()
	if err != nil {
		return err
	}
	return p.cp.ChangeMap(m, characterId, id)
}
