package rates

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type Processor interface {
	GetForCharacter(ch channel.Model, characterId uint32) Model
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetForCharacter(ch channel.Model, characterId uint32) Model {
	return GetForCharacter(p.l)(p.ctx)(ch, characterId)
}
