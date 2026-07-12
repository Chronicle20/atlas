package equipment

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(id uint32) (RestModel, error)
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

// GetById fetches equipment data from atlas-data by template ID
func (p *ProcessorImpl) GetById(id uint32) (RestModel, error) {
	return requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestById(id), func(rm RestModel) (RestModel, error) {
		return rm, nil
	})()
}
