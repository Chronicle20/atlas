package inventory

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetInventory(characterId uint32) (RestModel, error)
	GetAssets(characterId uint32, compartmentId string) ([]AssetRestModel, error)
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

// GetInventory retrieves a character's inventory from atlas-inventory
func (p *ProcessorImpl) GetInventory(characterId uint32) (RestModel, error) {
	return requestInventory(characterId)(p.l, p.ctx)
}

// GetAssets retrieves assets from a specific compartment
func (p *ProcessorImpl) GetAssets(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
	return requestAssets(characterId, compartmentId)(p.l, p.ctx)
}
