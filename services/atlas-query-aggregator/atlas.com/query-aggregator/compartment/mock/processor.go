package mock

import (
	"atlas-query-aggregator/compartment"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// ProcessorImpl is a mock implementation of the compartment.Processor interface
type ProcessorImpl struct {
	ByCharacterIdAndTypeProviderFunc func(characterId uint32, inventoryType inventory.Type) model.Provider[compartment.Model]
	GetByTypeFunc                    func(characterId uint32, inventoryType inventory.Type) (compartment.Model, error)
}

// ByCharacterIdAndTypeProvider returns a provider for compartment by character ID and type
func (m *ProcessorImpl) ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[compartment.Model] {
	if m.ByCharacterIdAndTypeProviderFunc != nil {
		return m.ByCharacterIdAndTypeProviderFunc(characterId, inventoryType)
	}
	return func() (compartment.Model, error) {
		return compartment.NewBuilder(uuid.New(), characterId, inventoryType, 100).Build(), nil
	}
}

// GetByType returns the compartment for a character and inventory type
func (m *ProcessorImpl) GetByType(characterId uint32, inventoryType inventory.Type) (compartment.Model, error) {
	if m.GetByTypeFunc != nil {
		return m.GetByTypeFunc(characterId, inventoryType)
	}
	return m.ByCharacterIdAndTypeProvider(characterId, inventoryType)()
}
