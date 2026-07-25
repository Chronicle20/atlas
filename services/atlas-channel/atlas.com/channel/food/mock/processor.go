package mock

import (
	"atlas-channel/food"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	RequestFeedFunc func(f field.Model, characterId character.Id, slot int16, itemId uint32) error
}

var _ food.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestFeed(f field.Model, characterId character.Id, slot int16, itemId uint32) error {
	if m.RequestFeedFunc != nil {
		return m.RequestFeedFunc(f, characterId, slot, itemId)
	}
	return nil
}
