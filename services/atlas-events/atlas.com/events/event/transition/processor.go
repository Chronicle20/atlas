package transition

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor is the generic entry point onto transition reads. Transitions
// are written only inside the occurrence administrator's paired
// occurrence+transition transaction (FR-O6/FR-T2, Task 15) — this package
// owns no write path, so Processor exposes reads only. Adding a write method
// here would open a second path to the same table; that is deliberately not
// done.
type Processor interface {
	GetByOccurrenceId(occurrenceId uuid.UUID) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetByOccurrenceId returns the full transition history for an occurrence,
// ordered oldest-first (FR-API5).
func (p *ProcessorImpl) GetByOccurrenceId(occurrenceId uuid.UUID) ([]Model, error) {
	return model.SliceMap(Make)(ByOccurrenceProvider(occurrenceId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}
