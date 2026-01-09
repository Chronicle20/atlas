package buddy

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for buddy list processing
type Processor interface {
	GetBuddyList(characterId uint32) model.Provider[Model]
	GetBuddyCapacity(characterId uint32) model.Provider[byte]
}

// processor implements the Processor interface
type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new buddy processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

// GetBuddyList returns the buddy list data for a character
func (p *processor) GetBuddyList(characterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		buddyProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
		buddyList, err := buddyProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get buddy list for character %d", characterId)
			return NewModel(characterId, 0), err
		}
		return buddyList, nil
	}
}

// GetBuddyCapacity returns the buddy list capacity for a character
func (p *processor) GetBuddyCapacity(characterId uint32) model.Provider[byte] {
	return func() (byte, error) {
		buddyProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
		buddyList, err := buddyProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get buddy capacity for character %d", characterId)
			return 0, err
		}
		return buddyList.Capacity(), nil
	}
}
