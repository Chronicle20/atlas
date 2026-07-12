package holding

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
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

func (p *ProcessorImpl) GetByCharacterProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacter(characterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return p.GetByCharacterProvider(characterId)()
}
