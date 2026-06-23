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
// serial->wish-entry resolution the CANCEL_WISH arm needs: the wish view renders
// each entry's per-(tenant, world) serial into the ITCITEM's nITCSN, and the
// client echoes that serial back on CANCEL_WISH. Writes (add/remove) go through
// the Kafka command processor, never this REST surface.
type Processor interface {
	GetByCharacterProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacter(characterId uint32) ([]Model, error)
	// GetByCharacterItem returns the character's wish entry for the given item
	// template. atlas-mts allows at most one wish per (character, item); an empty
	// result is reported as an error so the caller writes the matching *Failed.
	GetByCharacterItem(characterId uint32, itemId uint32) (Model, error)
	// GetByCharacterSerial returns the character's wish entry whose ITC serial
	// (nITCSN) matches the value the client echoed on CANCEL_WISH. An unmatched
	// serial is reported as an error so the caller writes CancelWishFailed.
	GetByCharacterSerial(characterId uint32, serial uint32) (Model, error)
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

func (p *ProcessorImpl) GetByCharacterSerial(characterId uint32, serial uint32) (Model, error) {
	ms, err := p.GetByCharacter(characterId)
	if err != nil {
		return Model{}, err
	}
	m, ok := findBySerial(ms, serial)
	if !ok {
		return Model{}, fmt.Errorf("character [%d] has no wish entry for serial [%d]", characterId, serial)
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

// findBySerial returns the wish entry whose ITC serial matches. A wish serial is
// unique per-(tenant, world) and a character is in exactly one world, so within a
// character's wishlist the serial identifies a single entry. Serial 0 never
// matches a real wish (serials start at 1), so a stale itcSn=0 row resolves to no
// entry rather than the wrong one.
func findBySerial(ms []Model, serial uint32) (Model, bool) {
	if serial == 0 {
		return Model{}, false
	}
	for _, m := range ms {
		if m.Serial() == serial {
			return m, true
		}
	}
	return Model{}, false
}
