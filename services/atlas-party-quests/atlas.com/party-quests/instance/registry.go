package instance

import (
	"errors"
	"sync"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("instance not found")

type Registry struct {
	lock       sync.Mutex
	instances  map[tenant.Model]map[uuid.UUID]Model
	tenantLock map[tenant.Model]*sync.RWMutex
	// Character-to-instance index for O(1) lookups
	characterToInstance map[tenant.Model]map[uint32]uuid.UUID
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{
			instances:           make(map[tenant.Model]map[uuid.UUID]Model),
			tenantLock:          make(map[tenant.Model]*sync.RWMutex),
			characterToInstance: make(map[tenant.Model]map[uint32]uuid.UUID),
		}
	})
	return registry
}

func (r *Registry) ensureTenant(t tenant.Model) *sync.RWMutex {
	if tl, ok := r.tenantLock[t]; ok {
		return tl
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if tl, ok := r.tenantLock[t]; ok {
		return tl
	}
	tl := &sync.RWMutex{}
	r.instances[t] = make(map[uuid.UUID]Model)
	r.tenantLock[t] = tl
	r.characterToInstance[t] = make(map[uint32]uuid.UUID)
	return tl
}

func (r *Registry) Create(t tenant.Model, m Model) Model {
	tl := r.ensureTenant(t)
	tl.Lock()
	defer tl.Unlock()

	r.instances[t][m.Id()] = m

	for _, c := range m.Characters() {
		r.characterToInstance[t][c.CharacterId] = m.Id()
	}

	return m
}

func (r *Registry) Get(t tenant.Model, instanceId uuid.UUID) (Model, error) {
	tl := r.ensureTenant(t)
	tl.RLock()
	defer tl.RUnlock()

	if m, ok := r.instances[t][instanceId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

func (r *Registry) GetByCharacter(t tenant.Model, characterId uint32) (Model, error) {
	tl := r.ensureTenant(t)
	tl.RLock()
	defer tl.RUnlock()

	if instanceId, ok := r.characterToInstance[t][characterId]; ok {
		if m, ok := r.instances[t][instanceId]; ok {
			return m, nil
		}
	}
	return Model{}, ErrNotFound
}

func (r *Registry) GetByMap(t tenant.Model, mapId uint32) []Model {
	tl := r.ensureTenant(t)
	tl.RLock()
	defer tl.RUnlock()

	var results []Model
	for _, m := range r.instances[t] {
		// Check if any character in this instance is associated with a map
		// The actual map tracking is done at a higher level via definitions/stages
		results = append(results, m)
	}
	return results
}

func (r *Registry) GetAll(t tenant.Model) []Model {
	tl := r.ensureTenant(t)
	tl.RLock()
	defer tl.RUnlock()

	results := make([]Model, 0, len(r.instances[t]))
	for _, m := range r.instances[t] {
		results = append(results, m)
	}
	return results
}

func (r *Registry) Update(t tenant.Model, instanceId uuid.UUID, updaters ...func(m Model) Model) (Model, error) {
	tl := r.ensureTenant(t)
	tl.Lock()
	defer tl.Unlock()

	oldModel, ok := r.instances[t][instanceId]
	if !ok {
		return Model{}, ErrNotFound
	}

	newModel := oldModel
	for _, updater := range updaters {
		newModel = updater(newModel)
	}

	// Update character index
	r.updateCharacterIndex(t, oldModel, newModel)

	r.instances[t][instanceId] = newModel
	return newModel, nil
}

func (r *Registry) Remove(t tenant.Model, instanceId uuid.UUID) {
	tl := r.ensureTenant(t)
	tl.Lock()
	defer tl.Unlock()

	if m, ok := r.instances[t][instanceId]; ok {
		for _, c := range m.Characters() {
			delete(r.characterToInstance[t], c.CharacterId)
		}
		delete(r.instances[t], instanceId)
	}
}

func (r *Registry) Clear(t tenant.Model) {
	tl := r.ensureTenant(t)
	tl.Lock()
	defer tl.Unlock()

	r.instances[t] = make(map[uuid.UUID]Model)
	r.characterToInstance[t] = make(map[uint32]uuid.UUID)
}

func (r *Registry) updateCharacterIndex(t tenant.Model, oldModel, newModel Model) {
	oldChars := make(map[uint32]bool)
	for _, c := range oldModel.Characters() {
		oldChars[c.CharacterId] = true
	}

	newChars := make(map[uint32]bool)
	for _, c := range newModel.Characters() {
		newChars[c.CharacterId] = true
	}

	for cid := range oldChars {
		if !newChars[cid] {
			delete(r.characterToInstance[t], cid)
		}
	}

	for cid := range newChars {
		if !oldChars[cid] {
			r.characterToInstance[t][cid] = newModel.Id()
		}
	}
}
