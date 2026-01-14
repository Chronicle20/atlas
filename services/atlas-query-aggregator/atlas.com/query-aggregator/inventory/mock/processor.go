package mock

import (
	"atlas-query-aggregator/inventory"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the inventory.Processor interface
type ProcessorImpl struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[inventory.Model]
	GetByCharacterIdFunc      func(characterId uint32) (inventory.Model, error)
}

// ByCharacterIdProvider returns a provider for inventory by character ID
func (m *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[inventory.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return func() (inventory.Model, error) {
		return inventory.NewBuilder(characterId).Build(), nil
	}
}

// GetByCharacterId returns the inventory for a character
func (m *ProcessorImpl) GetByCharacterId(characterId uint32) (inventory.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return m.ByCharacterIdProvider(characterId)()
}
