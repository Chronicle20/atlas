package transaction

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor is the channel-side read client for a character's atlas-mts
// transaction history. It backs the My Page -> History view (ITC section 4 /
// sub 2). Transaction rows are written server-side at settle, never through
// this REST surface.
type Processor interface {
	GetByCharacterProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacter(characterId uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// GetByCharacterProvider fetches every transaction-history row for a
// character. The upstream atlas-mts list is now paginated (task-117); the My
// Page -> History view renders the complete set, so this drains every page
// (up to page[size]=250 per request) rather than fetching just the first.
func (p *ProcessorImpl) GetByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byCharacterUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return p.GetByCharacterProvider(characterId)()
}
