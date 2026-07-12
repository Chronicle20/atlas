package transaction

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor exposes the read + create operations over MTS transaction-history
// rows. There is no saga: a transaction is a settled-fact record. Create is
// invoked by the settle handler (inside its existing custody transaction);
// GetByCharacter backs the My Page -> History read endpoint.
type Processor interface {
	GetByCharacter(characterId uint32) ([]Model, error)
	Create(m Model) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

// GetByCharacter returns all of a character's transaction-history rows,
// newest-first.
func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByCharacter(characterId)(p.db.WithContext(p.ctx)))()()
}

// GetAll resolves every transaction visible to the request's tenant.
func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()()
}

// Create persists a new transaction-history row and returns the stored Model.
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateTransaction(p.db.WithContext(p.ctx), m)
}
