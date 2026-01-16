package projection

import (
	"sync"
)

// ManagerInterface defines the interface for projection management
type ManagerInterface interface {
	Get(characterId uint32) (Model, bool)
	Create(characterId uint32, projection Model)
	Delete(characterId uint32)
	Update(characterId uint32, updateFn func(Model) Model) bool
}

// Manager is a singleton manager for storage projections keyed by characterId
type Manager struct {
	data sync.Map
}

var manager ManagerInterface
var managerOnce sync.Once

// GetManager returns the singleton instance of the projection manager
func GetManager() ManagerInterface {
	managerOnce.Do(func() {
		manager = &Manager{}
	})
	return manager
}

// Get retrieves a projection for a character
func (m *Manager) Get(characterId uint32) (Model, bool) {
	value, ok := m.data.Load(characterId)
	if !ok {
		return Model{}, false
	}
	return value.(Model), true
}

// Create stores a projection for a character
func (m *Manager) Create(characterId uint32, projection Model) {
	m.data.Store(characterId, projection)
}

// Delete removes a projection for a character
func (m *Manager) Delete(characterId uint32) {
	m.data.Delete(characterId)
}

// Update atomically updates a projection using the provided function.
// Returns true if the projection existed and was updated, false otherwise.
func (m *Manager) Update(characterId uint32, updateFn func(Model) Model) bool {
	value, ok := m.data.Load(characterId)
	if !ok {
		return false
	}
	updated := updateFn(value.(Model))
	m.data.Store(characterId, updated)
	return true
}
