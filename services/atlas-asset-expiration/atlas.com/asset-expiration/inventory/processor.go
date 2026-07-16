package inventory

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// identity is the no-op transformer for requests.DrainProvider, since
// AssetRestModel is already the target type for this consumer.
func identity(m AssetRestModel) (AssetRestModel, error) {
	return m, nil
}

// GetAssets retrieves ALL assets from a specific compartment. The upstream
// list is now paginated server-side (task-117); expiration checks must see
// every asset in the compartment, so this drains every page rather than
// fetching just the first.
func (p *ProcessorImpl) GetAssets(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
	return requests.DrainProvider[AssetRestModel, AssetRestModel](p.l, p.ctx)(compartmentAssetsUrl(characterId, compartmentId), 250, identity, model.Filters[AssetRestModel]())()
}
