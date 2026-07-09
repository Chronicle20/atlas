package record

import (
	"context"

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
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return GetByCharacter(p.ctx, p.db.WithContext(p.ctx), characterId)
}
