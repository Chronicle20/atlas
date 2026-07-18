package listing

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// browseAllPageSize is the page size requests.DrainProvider requests per
// iteration for the semantic-all BrowseAll path — paginate.MaxPageSize (250),
// matching the repo-wide DrainProvider convention (docs/rest-pagination.md
// §7) rather than atlas-mts's 16-per-page game default.
const browseAllPageSize = 250

// Processor is the channel-side read client for atlas-mts marketplace listings. It
// backs the browse (GET_ITC_LIST / SEARCH_ITC_LIST) ITC_OPERATION arms, which are
// synchronous reads — the channel queries atlas-mts REST and writes the
// GetItcListDone result inline (no status event). Writes (create/cancel/buy/bid/
// take-home) go through the Kafka command processor, never this REST surface.
type Processor interface {
	// BrowseProvider/Browse fetch exactly ONE page (the filter's Page/PageSize,
	// defaulting to atlas-mts's page 0 / 16-per-page) — the player-facing browse
	// (GET_ITC_LIST/SEARCH_ITC_LIST) and GetBySerial paths, where the caller
	// wants a bounded single request, not the complete matching set.
	BrowseProvider(worldId world.Id, f BrowseFilter) model.Provider[[]Model]
	Browse(worldId world.Id, f BrowseFilter) ([]Model, error)
	// BrowseAllProvider/BrowseAll drain every page of the filtered browse (the
	// semantic-all call sites: "my sales" panel, want-ad offers, cart favorites,
	// bidder/auction re-push), which must see every matching row regardless of
	// atlas-mts's per-page default. f.Page/f.PageSize are ignored — DrainProvider
	// owns paging.
	BrowseAllProvider(worldId world.Id, f BrowseFilter) model.Provider[[]Model]
	BrowseAll(worldId world.Id, f BrowseFilter) ([]Model, error)
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

// BrowseAllProvider drains every page of the filtered browse via
// requests.DrainProvider against the filter-only URL (browseUrl never bakes
// in Page/PageSize; DrainProvider appends its own page[number]/page[size]
// each iteration). This is the replacement for the removed atlas-mts
// PageSize:-1 "unpaged" escape hatch — the semantic-all call sites (my
// sales, want-ad offers, cart favorites, bidder auction re-push) need the
// complete matching set, not one server page.
func (p *ProcessorImpl) BrowseAllProvider(worldId world.Id, f BrowseFilter) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(browseUrl(worldId, f), browseAllPageSize, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) BrowseAll(worldId world.Id, f BrowseFilter) ([]Model, error) {
	return p.BrowseAllProvider(worldId, f)()
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
