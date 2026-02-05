package character

import (
	"atlas-rates/rate"
	"errors"
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

var ErrNotFound = errors.New("not found")

// Registry is the singleton in-memory cache for character rates
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

// Get retrieves a character's rate model
func (r *Registry) Get(t tenant.Model, characterId uint32) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	if m, ok := cm[characterId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

// GetOrCreate retrieves a character's rate model, creating one if it doesn't exist
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

// Update replaces a character's rate model
func (r *Registry) Update(t tenant.Model, m Model) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	cm[m.characterId] = m
}

// AddFactor adds or updates a rate factor for a character
func (r *Registry) AddFactor(t tenant.Model, ch channel.Model, characterId uint32, f rate.Factor) Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	var m Model
	var ok bool
	if m, ok = cm[characterId]; !ok {
		m = NewModel(t, ch, characterId)
	}

	m = m.WithFactor(f)
	cm[characterId] = m
	return m
}

// RemoveFactor removes a specific rate factor for a character
func (r *Registry) RemoveFactor(t tenant.Model, characterId uint32, source string, rateType rate.Type) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	m, ok := cm[characterId]
	if !ok {
		return Model{}, ErrNotFound
	}

	m = m.WithoutFactor(source, rateType)
	cm[characterId] = m
	return m, nil
}

// RemoveFactorsBySource removes all factors from a specific source for a character
func (r *Registry) RemoveFactorsBySource(t tenant.Model, characterId uint32, source string) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	m, ok := cm[characterId]
	if !ok {
		return Model{}, ErrNotFound
	}

	m = m.WithoutFactorsBySource(source)
	cm[characterId] = m
	return m, nil
}

// GetAllForWorld returns all characters in a specific world
func (r *Registry) GetAllForWorld(t tenant.Model, worldId world.Id) []Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	result := make([]Model, 0)
	for _, m := range cm {
		if m.worldId == worldId {
			result = append(result, m)
		}
	}
	return result
}

// UpdateWorldRate updates the world rate factor for all characters in that world
func (r *Registry) UpdateWorldRate(t tenant.Model, worldId world.Id, rateType rate.Type, multiplier float64) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	source := "world"
	f := rate.NewFactor(source, rateType, multiplier)

	for id, m := range cm {
		if m.worldId == worldId {
			cm[id] = m.WithFactor(f)
		}
	}
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
