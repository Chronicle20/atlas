package ledger

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor is the ledger's application-layer face. The REST resource and the
// settlement handler go through it rather than reaching into the provider and
// administrator functions directly (DOM-14).
type Processor interface {
	// Record durably writes a settled trade. It is idempotent per settlement
	// transaction: recording the same entry twice returns the entry that was
	// already stored (FR-5.7).
	Record(m Model) (Model, error)

	// GetById returns one ledger entry.
	GetById(id uuid.UUID) (Model, error)

	// GetByCharacterId returns every entry settled in [from, to] on which the
	// character appears as either side, newest first (FR-7.2).
	GetByCharacterId(characterId character.Id, from time.Time, to time.Time) ([]Model, error)

	// GetPageByCharacterId returns one page of the same selection, paged in the
	// database. The REST list read uses this rather than GetByCharacterId so a
	// busy character's whole history is never materialised to serve one page.
	GetPageByCharacterId(characterId character.Id, from time.Time, to time.Time, page model.Page) (model.Paged[Model], error)
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

func (p *ProcessorImpl) Record(m Model) (Model, error) {
	return create(p.db.WithContext(p.ctx), p.t.Id())(m)
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return byId(p.db.WithContext(p.ctx), p.t.Id())(id)
}

func (p *ProcessorImpl) GetByCharacterId(characterId character.Id, from time.Time, to time.Time) ([]Model, error) {
	return byCharacter(p.db.WithContext(p.ctx), p.t.Id())(characterId, from, to)
}

func (p *ProcessorImpl) GetPageByCharacterId(characterId character.Id, from time.Time, to time.Time, page model.Page) (model.Paged[Model], error) {
	return pageByCharacter(p.db.WithContext(p.ctx), p.t.Id())(characterId, from, to, page)
}
