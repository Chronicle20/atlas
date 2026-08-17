package location

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the character-location REST client used by the invite path.
type Processor interface {
	FieldOf(characterId character.Id) (field.Model, error)
	FieldProvider(characterId character.Id) model.Provider[field.Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) FieldProvider(characterId character.Id) model.Provider[field.Model] {
	return requests.Provider[RestModel, field.Model](p.l, p.ctx)(requestByCharacterId(p.ctx, characterId), Extract)
}

// FieldOf returns the field atlas-maps has the character standing in. A 404
// (no stored location) is returned as an error like any other failure — the
// invite path refuses rather than guessing where the target is.
func (p *ProcessorImpl) FieldOf(characterId character.Id) (field.Model, error) {
	return p.FieldProvider(characterId)()
}

// Extract folds the wire model into the shared field type.
func Extract(rm RestModel) (field.Model, error) {
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}
