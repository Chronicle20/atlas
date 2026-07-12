package bid

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor exposes the REST-facing operations over auction bids. Kafka
// emission (the MethodAndEmit convention) lands in a later phase; for now these
// are REST-only reads/writes.
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	Create(m Model) (Model, error)
	GetByListingId(listingId uuid.UUID) ([]Model, error)
	TransitionState(id string, from State, to State) (bool, error)
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

// Create persists a new bid and returns the stored Model (with its assigned
// surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateBid(p.db.WithContext(p.ctx), m)
}

// GetByListingId returns the bids placed on an auction listing. The signature
// mirrors the getByListingId provider exactly.
func (p *ProcessorImpl) GetByListingId(listingId uuid.UUID) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByListingId(listingId)(p.db.WithContext(p.ctx)))()()
}

// TransitionState performs the race-safe conditional transition, returning true
// iff exactly one row moved from `from` to `to`.
func (p *ProcessorImpl) TransitionState(id string, from State, to State) (bool, error) {
	affected, err := UpdateState(p.db.WithContext(p.ctx), id, from, to)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
