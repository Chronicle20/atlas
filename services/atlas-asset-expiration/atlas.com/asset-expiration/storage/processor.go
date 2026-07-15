package storage

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// identity is the no-op transformer for requests.DrainProvider, since
// AssetRestModel is already the target type for this consumer.
func identity(m AssetRestModel) (AssetRestModel, error) {
	return m, nil
}

// GetAssets retrieves ALL assets from storage. The upstream list is now
// paginated server-side (task-117); expiration checks must see every asset
// in the storage, so this drains every page rather than fetching just the
// first.
func (p *ProcessorImpl) GetAssets(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
	return requests.DrainProvider[AssetRestModel, AssetRestModel](p.l, p.ctx)(storageAssetsUrl(accountId, worldId), 250, identity, model.Filters[AssetRestModel]())()
}
