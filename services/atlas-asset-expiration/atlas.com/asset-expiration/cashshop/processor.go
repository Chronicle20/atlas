package cashshop

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetCompartments(accountId uint32) ([]CompartmentRestModel, error)
	GetAllItems(accountId uint32) ([]ItemRestModel, error)
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

// GetCompartments retrieves all cash shop compartments from atlas-cashshop
func (p *ProcessorImpl) GetCompartments(accountId uint32) ([]CompartmentRestModel, error) {
	return requestCompartments(accountId)(p.l, p.ctx)
}

// GetAllItems retrieves all items across all compartments for an account
func (p *ProcessorImpl) GetAllItems(accountId uint32) ([]ItemRestModel, error) {
	comps, err := p.GetCompartments(accountId)
	if err != nil {
		return nil, err
	}

	var allItems []ItemRestModel
	for _, comp := range comps {
		for _, asset := range comp.Assets {
			allItems = append(allItems, asset.Item)
		}
	}

	return allItems, nil
}
