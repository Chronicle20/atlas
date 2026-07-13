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
	// ByCharacterPagedProvider returns one page of a character's transaction
	// history, newest-first (the REST list handler, task-117).
	ByCharacterPagedProvider(characterId uint32, page model.Page) model.Provider[model.Paged[Model]]
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

// Create persists a new transaction-history row and returns the stored Model.
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateTransaction(p.db.WithContext(p.ctx), m)
}

// ByCharacterPagedProvider returns one page of a character's transaction
// history, newest-first.
func (p *ProcessorImpl) ByCharacterPagedProvider(characterId uint32, page model.Page) model.Provider[model.Paged[Model]] {
	return model.MapPaged(modelFromEntity)(getByCharacterPaged(characterId, page)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}
