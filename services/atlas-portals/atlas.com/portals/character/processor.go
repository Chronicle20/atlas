package character

import (
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	EnableActions(f field.Model, characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) EnableActions(f field.Model, characterId uint32) {
	_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicCharacterStatus)(enableActionsProvider(f, characterId))
}
