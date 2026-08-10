package settlement

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor is the settlement record's application-layer face.
type Processor interface {
	// Submit durably records a settlement saga this service has just
	// submitted. It must be called inside the transaction that enqueues the
	// saga command, so the record and the command cannot diverge.
	Submit(m Model) (Model, error)

	// GetByTransactionId returns the unresolved settlement with the given saga
	// transaction id.
	GetByTransactionId(transactionId uuid.UUID) (Model, error)

	// Resolve deletes the record once its terminal status has been handled and
	// reports whether THIS call is the one that deleted it. That boolean is the
	// terminal path's arbiter: two concurrent deliveries serialise on the
	// delete and exactly one is told it won, so exactly one emits the outcome.
	// A second call simply reports false, which is what makes the terminal path
	// safe to run twice.
	Resolve(transactionId uuid.UUID) (bool, error)

	// Unresolved returns every unfinished settlement for the request's tenant,
	// oldest first.
	Unresolved() ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

// NewProcessor resolves the tenant from ctx once; every query the processor
// issues is filtered on that tenant id.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Submit(m Model) (Model, error) {
	return create(p.db.WithContext(p.ctx), p.t)(m)
}

func (p *ProcessorImpl) GetByTransactionId(transactionId uuid.UUID) (Model, error) {
	return byTransactionId(p.db.WithContext(p.ctx), p.t.Id())(transactionId)
}

func (p *ProcessorImpl) Resolve(transactionId uuid.UUID) (bool, error) {
	return deleteByTransactionId(p.db.WithContext(p.ctx), p.t.Id())(transactionId)
}

func (p *ProcessorImpl) Unresolved() ([]Model, error) {
	return unresolvedForTenant(p.db.WithContext(p.ctx), p.t.Id())
}

// Unresolved returns every unfinished settlement across every tenant, oldest
// first, for startup reconciliation.
//
// It is a package function rather than a Processor method because there is no
// tenant to construct a Processor with at boot: each returned Model carries the
// tenant it belongs to, and the caller restores it per row via Model.Tenant().
func Unresolved(ctx context.Context, db *gorm.DB) ([]Model, error) {
	return allUnresolved(db.WithContext(ctx))
}
