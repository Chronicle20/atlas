package holding

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor exposes the REST-facing operations over take-home holdings. Kafka
// emission (the MethodAndEmit convention) lands in a later phase; for now these
// are REST-only reads/writes.
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	Create(m Model) (Model, error)
	GetByOwner(worldId world.Id, ownerId uint32) ([]Model, error)
	GetByCharacter(ownerId uint32) ([]Model, error)
	TakeHome(id string) (bool, error)
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

// Create persists a new holding and returns the stored Model (with its assigned
// surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateHolding(p.db.WithContext(p.ctx), m)
}

// GetByOwner returns the holdings for a character in a world. The signature
// mirrors the getByOwner provider exactly.
func (p *ProcessorImpl) GetByOwner(worldId world.Id, ownerId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByOwner(worldId, ownerId)(p.db.WithContext(p.ctx)))()()
}

// GetByCharacter returns all holdings for a character (owner) across worlds. The
// signature mirrors the getByCharacter provider exactly.
func (p *ProcessorImpl) GetByCharacter(ownerId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByCharacter(ownerId)(p.db.WithContext(p.ctx)))()()
}

// TakeHome soft-deletes the holding by id, returning true iff exactly one row
// was soft-deleted. A repeated take-home is idempotent: the second call returns
// false because the row is already gone from the default scope.
func (p *ProcessorImpl) TakeHome(id string) (bool, error) {
	affected, err := SoftDelete(p.db.WithContext(p.ctx), id)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
