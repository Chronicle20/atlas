package purchaserecord

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor scopes Record/Get to the calling tenant so downstream callers
// (e.g. GET_PURCHASE_RECORD) do not have to thread tenantId through by hand.
type Processor interface {
	Record(db *gorm.DB, accountId uint32, serialNumber uint32) error
	Get(accountId uint32, serialNumber uint32) (uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Record upserts a purchase on the caller-supplied handle. Callers inside an
// existing transaction pass the tx; the db field on ProcessorImpl is only
// used by Get below.
func (p *ProcessorImpl) Record(db *gorm.DB, accountId uint32, serialNumber uint32) error {
	return Record(db, p.t.Id(), accountId, serialNumber)
}

func (p *ProcessorImpl) Get(accountId uint32, serialNumber uint32) (uint32, error) {
	return Get(p.db.WithContext(p.ctx), p.t.Id(), accountId, serialNumber)
}
