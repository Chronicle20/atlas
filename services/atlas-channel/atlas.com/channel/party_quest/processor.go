package party_quest

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetTimerByCharacterId(characterId uint32) (TimerModel, error)
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

func (p *ProcessorImpl) GetTimerByCharacterId(characterId uint32) (TimerModel, error) {
	return requests.Provider[TimerRestModel, TimerModel](p.l, p.ctx)(requestTimerByCharacterId(characterId), ExtractTimer)()
}
