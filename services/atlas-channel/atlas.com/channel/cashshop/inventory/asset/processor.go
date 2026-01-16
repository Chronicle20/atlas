package asset

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByIdProvider(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) model.Provider[Model]
	GetById(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) (Model, error)
	ByCompartmentIdProvider(accountId uint32, compartmentId uuid.UUID) model.Provider[[]Model]
	GetByCompartmentId(accountId uint32, compartmentId uuid.UUID) ([]Model, error)
	GetByItemId(accountId uint32, compartmentId uuid.UUID, itemId uint32) (Model, error)
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

// ByCompartmentIdProvider returns a provider function that fetches all assets for a compartment
func (p *ProcessorImpl) ByCompartmentIdProvider(accountId uint32, compartmentId uuid.UUID) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCompartmentId(accountId, compartmentId), Extract, model.Filters[Model]())
}

// GetByCompartmentId retrieves all assets for a compartment
func (p *ProcessorImpl) GetByCompartmentId(accountId uint32, compartmentId uuid.UUID) ([]Model, error) {
	return p.ByCompartmentIdProvider(accountId, compartmentId)()
}

// GetByItemId retrieves an asset by item ID within a compartment
func (p *ProcessorImpl) GetByItemId(accountId uint32, compartmentId uuid.UUID, itemId uint32) (Model, error) {
	assets, err := p.GetByCompartmentId(accountId, compartmentId)
	if err != nil {
		return Model{}, err
	}
	for _, a := range assets {
		if a.Item().Id() == itemId {
			return a, nil
		}
	}
	return Model{}, fmt.Errorf("asset with item ID [%d] not found in compartment [%s]", itemId, compartmentId)
}