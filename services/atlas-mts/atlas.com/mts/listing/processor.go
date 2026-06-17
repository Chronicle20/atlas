package listing

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor exposes the REST-facing CRUD and state-transition operations over
// marketplace listings. Kafka emission (the MethodAndEmit convention) lands in
// Phase 3; for now these are REST-only reads/writes.
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	Create(m Model) (Model, error)
	Browse(worldId world.Id, state State, f BrowseFilter) ([]Model, error)
	TransitionState(id string, from State, to State) (bool, error)
	UpdateAuction(id string, currentBid uint32, highBidderId uint32, endsAt *time.Time) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetById(id string) (Model, error) {
	return GetById(id)(p.db.WithContext(p.ctx))()
}

// Create persists a new listing and returns the stored Model (with its assigned
// surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateListing(p.db.WithContext(p.ctx), m)
}

// Browse returns the listings for a world filtered by state and the optional
// filter set (category, sub-category, sale type, item id, seller name) with
// pagination. The signature mirrors the getBrowse provider exactly.
func (p *ProcessorImpl) Browse(worldId world.Id, state State, f BrowseFilter) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getBrowse(worldId, state, f)(p.db.WithContext(p.ctx)))()()
}

// TransitionState performs the race-safe conditional transition, returning true
// iff exactly one row moved from `from` to `to` (the cancel-vs-buy race resolves
// to a single winner; a loser sees zero rows affected and gets false).
func (p *ProcessorImpl) TransitionState(id string, from State, to State) (bool, error) {
	affected, err := UpdateState(p.db.WithContext(p.ctx), id, from, to)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// UpdateAuction updates the live auction fields (current bid, high bidder, and
// the optional end time). Used by the bid path.
func (p *ProcessorImpl) UpdateAuction(id string, currentBid uint32, highBidderId uint32, endsAt *time.Time) error {
	return UpdateAuction(p.db.WithContext(p.ctx), id, currentBid, highBidderId, endsAt)
}
