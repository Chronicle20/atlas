package wishlist

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for wishlist processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	SetForCharacter(characterId uint32, serialNumbers []uint32) ([]Model, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// ByCharacterIdProvider fetches every wishlist item for a character. The
// upstream atlas-cashshop list is now paginated (task-117); callers here
// (character-info popup, cash-shop entry) need the complete wishlist, so
// this drains every page rather than fetching just the first.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byCharacterIdUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) SetForCharacter(characterId uint32, serialNumbers []uint32) ([]Model, error) {
	p.l.Debugf("Setting wishlist for character [%d].", characterId)
	results := make([]Model, 0)
	err := clearForCharacterId(characterId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to clear wishlist for character [%d].", characterId)
		return results, err
	}
	for _, serialNumber := range serialNumbers {
		if serialNumber == 0 {
			continue
		}
		var rm RestModel
		rm, err = addForCharacterId(characterId, serialNumber)(p.l, p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to add serialNumber [%d] to wishlist for character [%d].", serialNumber, characterId)
			continue
		}
		var m Model
		m, err = Extract(rm)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to extract wishlist item for character [%d].", characterId)
		}
		results = append(results, m)
	}
	return results, nil
}
