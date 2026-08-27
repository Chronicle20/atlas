package _map

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Processor interface {
	CharacterIdsInField(f field.Model) ([]uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) CharacterIdsInField(f field.Model) ([]uint32, error) {
	return CharacterIdsInFieldModelProvider(p.l)(p.ctx)(f)()
}
