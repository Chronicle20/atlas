package petdata

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor provides operations for querying pet evolution data from atlas-data
type Processor interface {
	GetById(petTemplateId uint32) (Model, error)
}

type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new pet data processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*processor)(nil)

// GetById retrieves pet evolution data for the given pet template id
func (p *processor) GetById(petTemplateId uint32) (Model, error) {
	m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petTemplateId), Extract)()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get pet data for template %d", petTemplateId)
		return Model{}, err
	}
	return m, nil
}
