package weather

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetActive(f field.Model) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetActive(f field.Model) (RestModel, error) {
	return requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestWeatherInMap(f), Extract)()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
