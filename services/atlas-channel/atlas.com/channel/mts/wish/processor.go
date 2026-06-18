package wish

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor is the channel-side read client for a character's atlas-mts wish list.
// It backs the synchronous VIEW_WISH arm (LoadWishSaleListDone) and the
// serial->wish-entry resolution the DELETE_ZZIM / CANCEL_WISH arms need (the wire
// carries a listing serial, never the wish UUID). Writes (add/remove) go through
// the Kafka command processor, never this REST surface.
type Processor interface {
	GetByCharacterProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacter(characterId uint32) ([]Model, error)
	// GetByCharacterItem returns the character's wish entry for the given item
	// template. atlas-mts allows at most one wish per (character, item); an empty
	// result is reported as an error so the caller writes the matching *Failed.
	GetByCharacterItem(characterId uint32, itemId uint32) (Model, error)
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

func (p *ProcessorImpl) GetByCharacterItem(characterId uint32, itemId uint32) (Model, error) {
	ms, err := p.GetByCharacter(characterId)
	if err != nil {
		return Model{}, err
	}
	m, ok := findByItem(ms, itemId)
	if !ok {
		return Model{}, fmt.Errorf("character [%d] has no wish entry for item [%d]", characterId, itemId)
	}
	return m, nil
}

// findByItem returns the first wish entry whose ItemId matches (atlas-mts allows
// at most one wish per (character, item), so the first match is the entry). It is
// the pure selection logic GetByCharacterItem applies to the REST result.
func findByItem(ms []Model, itemId uint32) (Model, bool) {
	for _, m := range ms {
		if m.ItemId() == itemId {
			return m, true
		}
	}
	return Model{}, false
}
