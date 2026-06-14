package mount

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
}

// GetByCharacterId returns the character's mount progression. A character with no
// mount record yields an error (no row); callers treat that as "no mount block".
func (p *Processor) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}
