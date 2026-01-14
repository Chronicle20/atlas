package mock

import (
	"atlas-query-aggregator/buddy"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the buddy.Processor interface
type ProcessorImpl struct {
	GetBuddyListFunc     func(characterId uint32) model.Provider[buddy.Model]
	GetBuddyCapacityFunc func(characterId uint32) model.Provider[byte]
}

// GetBuddyList returns the buddy list for a character
func (m *ProcessorImpl) GetBuddyList(characterId uint32) model.Provider[buddy.Model] {
	if m.GetBuddyListFunc != nil {
		return m.GetBuddyListFunc(characterId)
	}
	return func() (buddy.Model, error) {
		return buddy.NewModel(characterId, 20), nil
	}
}

// GetBuddyCapacity returns the buddy list capacity for a character
func (m *ProcessorImpl) GetBuddyCapacity(characterId uint32) model.Provider[byte] {
	if m.GetBuddyCapacityFunc != nil {
		return m.GetBuddyCapacityFunc(characterId)
	}
	return func() (byte, error) {
		return 20, nil
	}
}
