package listing

import (
	"context"
	"fmt"

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
	// GetBySerial resolves the single active listing carrying the given ITC serial
	// (the client's nITCSN). The wish/zzim ITC arms (SET_ZZIM/DELETE_ZZIM/
	// CANCEL_WISH) carry only the serial; this resolves it to the listing's
	// templateId so the channel can address the matching wish entry by item.
	GetBySerial(worldId world.Id, serial uint32) (Model, error)
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

// GetBySerial browses with the serial filter and returns the single matching
// active listing. atlas-mts's serial column is unique per (tenant, world) so the
// filtered browse returns at most one row; an empty result is reported as an
// error so the caller can write the matching *Failed result.
func (p *ProcessorImpl) GetBySerial(worldId world.Id, serial uint32) (Model, error) {
	ms, err := p.Browse(worldId, BrowseFilter{Serial: serial})
	if err != nil {
		return Model{}, err
	}
	if len(ms) == 0 {
		return Model{}, fmt.Errorf("no active listing for world [%d] serial [%d]", byte(worldId), serial)
	}
	return ms[0], nil
}
