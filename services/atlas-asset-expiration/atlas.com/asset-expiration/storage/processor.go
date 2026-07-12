package storage

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetAssets(accountId uint32, worldId world.Id) ([]AssetRestModel, error)
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

// GetAssets retrieves all assets from storage
func (p *ProcessorImpl) GetAssets(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
	return requestAssets(accountId, worldId)(p.l, p.ctx)
}
