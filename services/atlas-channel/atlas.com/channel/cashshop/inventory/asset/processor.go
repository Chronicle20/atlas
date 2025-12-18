package asset

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByIdProvider(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) model.Provider[Model]
	GetById(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) (Model, error)
}

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

// ByIdProvider returns a provider function that fetches an asset by ID
func (p *ProcessorImpl) ByIdProvider(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(accountId, compartmentId, assetId), Extract)
}

// GetById retrieves an asset by ID
func (p *ProcessorImpl) GetById(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) (Model, error) {
	return p.ByIdProvider(accountId, compartmentId, assetId)()
}