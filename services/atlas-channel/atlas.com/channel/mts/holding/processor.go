package holding

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the channel-side read client for a character's atlas-mts take-home
// holdings. It backs the ENTER_MTS holding announce (GET_USER_PURCHASE_ITEM_DONE).
// Writes (take-home) go through the Kafka command processor, never this REST
// surface.
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

// GetByCharacterProvider fetches every take-home holding for a character. The
// upstream atlas-mts list is now paginated (task-117); callers here (MTS entry
// announce, the post-take-home re-push) need the complete set, so this drains
// every page rather than fetching just the first.
func (p *ProcessorImpl) GetByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byCharacterUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return p.GetByCharacterProvider(characterId)()
}
