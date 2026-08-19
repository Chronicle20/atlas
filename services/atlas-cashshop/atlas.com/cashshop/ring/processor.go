package ring

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor scopes CreatePair/GetByCharacterId/GetById to the calling
// tenant so downstream callers (the ring purchase path, a future effects
// consumer) do not have to thread tenantId through by hand.
type Processor interface {
	CreatePair(db *gorm.DB, ringType Type, a Half, b Half) (uuid.UUID, error)
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetById(id uuid.UUID) (Model, error)
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

// CreatePair takes the caller-supplied handle -- it is called inside the
// ring purchase transaction, so it must run on that transaction's tx, not on
// the processor's own db.
func (p *ProcessorImpl) CreatePair(db *gorm.DB, ringType Type, a Half, b Half) (uuid.UUID, error) {
	return CreatePair(db, p.t.Id(), ringType, a, b)
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return GetByCharacterId(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return GetById(p.db.WithContext(p.ctx), p.t.Id(), id)
}
