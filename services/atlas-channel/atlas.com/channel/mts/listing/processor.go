package listing

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor is the channel-side read client for atlas-mts marketplace listings. It
// backs the browse (GET_ITC_LIST / SEARCH_ITC_LIST) ITC_OPERATION arms, which are
// synchronous reads — the channel queries atlas-mts REST and writes the
// GetItcListDone result inline (no status event). Writes (create/cancel/buy/bid/
// take-home) go through the Kafka command processor, never this REST surface.
type Processor interface {
	BrowseProvider(worldId world.Id, f BrowseFilter) model.Provider[[]Model]
	Browse(worldId world.Id, f BrowseFilter) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) BrowseProvider(worldId world.Id, f BrowseFilter) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestBrowse(worldId, f), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) Browse(worldId world.Id, f BrowseFilter) ([]Model, error) {
	return p.BrowseProvider(worldId, f)()
}
