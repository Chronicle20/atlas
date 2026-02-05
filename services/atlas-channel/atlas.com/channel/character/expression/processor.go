package expression

import (
	expression2 "atlas-channel/kafka/message/expression"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for expression processing
type Processor interface {
	Change(characterId uint32, f field.Model, expression uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) Change(characterId uint32, f field.Model, expression uint32) error {
	p.l.Debugf("Changing character [%d] expression to [%d].", characterId, f.MapId())
	return producer.ProviderImpl(p.l)(p.ctx)(expression2.EnvExpressionCommand)(SetCommandProvider(characterId, f, expression))
}
