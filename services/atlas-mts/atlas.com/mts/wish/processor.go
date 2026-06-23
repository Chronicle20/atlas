package wish

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor exposes the REST-facing operations over wish-list entries. Kafka
// emission (the MethodAndEmit convention) lands in a later phase; for now these
// are REST-only reads/writes.
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	GetBySerial(worldId world.Id, sn uint32) (Model, error)
	Create(m Model) (Model, error)
	GetByCharacter(characterId uint32) ([]Model, error)
	Delete(id string) (bool, error)
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

// GetBySerial resolves a wish entry by its per-(tenant, world) ITC serial.
func (p *ProcessorImpl) GetBySerial(worldId world.Id, sn uint32) (Model, error) {
	return GetBySerial(worldId, sn)(p.db.WithContext(p.ctx))()
}

// Create persists a new wish entry and returns the stored Model (with its
// assigned surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateWish(p.db.WithContext(p.ctx), m)
}

// GetByCharacter returns the wish entries for a character. The signature mirrors
// the getByCharacter provider exactly.
func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByCharacter(characterId)(p.db.WithContext(p.ctx)))()()
}

// Delete hard-deletes the wish entry by id, returning true iff exactly one row
// was deleted.
func (p *ProcessorImpl) Delete(id string) (bool, error) {
	affected, err := DeleteWish(p.db.WithContext(p.ctx), id)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
