package character

import (
	"atlas-effective-stats/stat"
	"errors"
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

var ErrNotFound = errors.New("not found")

// Registry is the singleton in-memory cache for character effective stats
type Registry struct {
	lock         sync.Mutex
	characterReg map[tenant.Model]map[uint32]Model
	tenantLock   map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

// GetRegistry returns the singleton registry instance
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

// getOrCreateTenantMaps returns the character map and lock for a tenant,
// creating them if they don't exist. This method is thread-safe.
func (r *Registry) getOrCreateTenantMaps(t tenant.Model) (map[uint32]Model, *sync.RWMutex) {
	r.lock.Lock()
	defer r.lock.Unlock()

	cm, ok := r.characterReg[t]
	if !ok {
		cm = make(map[uint32]Model)
		r.characterReg[t] = cm
	}

	cml, ok := r.tenantLock[t]
	if !ok {
		cml = &sync.RWMutex{}
		r.tenantLock[t] = cml
	}

	return cm, cml
}

// Get retrieves a character's effective stats model
func (r *Registry) Get(t tenant.Model, characterId uint32) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	if m, ok := cm[characterId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

// GetOrCreate retrieves a character's effective stats model, creating one if it doesn't exist
func (r *Registry) GetOrCreate(t tenant.Model, ch channel.Model, characterId uint32) Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	if m, ok := cm[characterId]; ok {
		return m
	}

	m := NewModel(t, ch, characterId)
	cm[characterId] = m
	return m
}

// Update replaces a character's effective stats model
func (r *Registry) Update(t tenant.Model, m Model) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	cm[m.characterId] = m
}

// AddBonus adds or updates a stat bonus for a character
func (r *Registry) AddBonus(t tenant.Model, ch channel.Model, characterId uint32, b stat.Bonus) Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	var m Model
	var ok bool
	if m, ok = cm[characterId]; !ok {
		m = NewModel(t, ch, characterId)
	}

	m = m.WithBonus(b).Recompute()
	cm[characterId] = m
	return m
}

// AddBonuses adds or updates multiple stat bonuses for a character
func (r *Registry) AddBonuses(t tenant.Model, ch channel.Model, characterId uint32, bonuses []stat.Bonus) Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	var m Model
	var ok bool
	if m, ok = cm[characterId]; !ok {
		m = NewModel(t, ch, characterId)
	}

	m = m.WithBonuses(bonuses).Recompute()
	cm[characterId] = m
	return m
}

// RemoveBonus removes a specific stat bonus for a character
func (r *Registry) RemoveBonus(t tenant.Model, characterId uint32, source string, statType stat.Type) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	m, ok := cm[characterId]
	if !ok {
		return Model{}, ErrNotFound
	}

	m = m.WithoutBonus(source, statType).Recompute()
	cm[characterId] = m
	return m, nil
}

// RemoveBonusesBySource removes all bonuses from a specific source for a character
func (r *Registry) RemoveBonusesBySource(t tenant.Model, characterId uint32, source string) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	m, ok := cm[characterId]
	if !ok {
		return Model{}, ErrNotFound
	}

	m = m.WithoutBonusesBySource(source).Recompute()
	cm[characterId] = m
	return m, nil
}

// SetBaseStats sets the base stats for a character and recomputes effective stats
func (r *Registry) SetBaseStats(t tenant.Model, ch channel.Model, characterId uint32, base stat.Base) Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	var m Model
	var ok bool
	if m, ok = cm[characterId]; !ok {
		m = NewModel(t, ch, characterId)
	}

	m = m.WithBaseStats(base).Recompute()
	cm[characterId] = m
	return m
}

// MarkInitialized marks a character as initialized
func (r *Registry) MarkInitialized(t tenant.Model, characterId uint32) error {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	m, ok := cm[characterId]
	if !ok {
		return ErrNotFound
	}

	m = m.WithInitialized()
	cm[characterId] = m
	return nil
}

// IsInitialized checks if a character has been initialized
func (r *Registry) IsInitialized(t tenant.Model, characterId uint32) bool {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	if m, ok := cm[characterId]; ok {
		return m.Initialized()
	}
	return false
}

// GetAll returns all characters for a tenant
func (r *Registry) GetAll(t tenant.Model) []Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	result := make([]Model, 0, len(cm))
	for _, m := range cm {
		result = append(result, m)
	}
	return result
}

// GetAllForWorld returns all characters in a specific world
func (r *Registry) GetAllForWorld(t tenant.Model, worldId world.Id) []Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	result := make([]Model, 0)
	for _, m := range cm {
		if m.WorldId() == worldId {
			result = append(result, m)
		}
	}
	return result
}

// Delete removes a character from the registry
func (r *Registry) Delete(t tenant.Model, characterId uint32) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	delete(cm, characterId)
}

// ResetForTesting clears all registry state (testing only)
func (r *Registry) ResetForTesting() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.characterReg = make(map[tenant.Model]map[uint32]Model)
	r.tenantLock = make(map[tenant.Model]*sync.RWMutex)
}
