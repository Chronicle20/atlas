package record

import (
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor is the record domain's application-layer face. The REST handler
// (resource.go) and any other caller go through it rather than reaching into
// the provider functions directly (DOM-14; buddies list.Processor shape,
// services/atlas-buddies/atlas.com/buddies/list/processor.go:25).
type Processor interface {
	// GetByCharacter returns one Model per GameType for the character,
	// zero-filled for any game type with no rows yet.
	GetByCharacter(characterId uint32) ([]Model, error)
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

func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return GetByCharacter(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}
