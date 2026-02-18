package party_quest

import (
	"context"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{
		l:   l,
		ctx: ctx,
	}
}

func (p *Processor) GetTimerByCharacterId(characterId uint32) (TimerModel, error) {
	return requests.Provider[TimerRestModel, TimerModel](p.l, p.ctx)(requestTimerByCharacterId(characterId), ExtractTimer)()
}
