package character

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
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
