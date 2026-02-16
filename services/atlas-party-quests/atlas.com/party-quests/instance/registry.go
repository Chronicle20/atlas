package instance

import (
	"errors"
	"sync"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("instance not found")

type tenantData struct {
	lock                sync.RWMutex
	instances           map[uuid.UUID]Model
	characterToInstance map[uint32]uuid.UUID
}

type Registry struct {
	lock    sync.Mutex
	tenants map[tenant.Model]*tenantData
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{
			tenants: make(map[tenant.Model]*tenantData),
		}
	})
	return registry
}

func (r *Registry) ensureTenant(t tenant.Model) *tenantData {
	r.lock.Lock()
	defer r.lock.Unlock()
	if td, ok := r.tenants[t]; ok {
		return td
	}
	td := &tenantData{
		instances:           make(map[uuid.UUID]Model),
		characterToInstance: make(map[uint32]uuid.UUID),
	}
	r.tenants[t] = td
	return td
}

func (r *Registry) Create(t tenant.Model, m Model) Model {
	td := r.ensureTenant(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	td.instances[m.Id()] = m

	for _, c := range m.Characters() {
		td.characterToInstance[c.CharacterId()] = m.Id()
	}

	return m
}

func (r *Registry) Get(t tenant.Model, instanceId uuid.UUID) (Model, error) {
	td := r.ensureTenant(t)
	td.lock.RLock()
	defer td.lock.RUnlock()

	if m, ok := td.instances[instanceId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

func (r *Registry) GetByCharacter(t tenant.Model, characterId uint32) (Model, error) {
	td := r.ensureTenant(t)
	td.lock.RLock()
	defer td.lock.RUnlock()

	if instanceId, ok := td.characterToInstance[characterId]; ok {
		if m, ok := td.instances[instanceId]; ok {
			return m, nil
		}
	}
	return Model{}, ErrNotFound
}

func (r *Registry) GetAll(t tenant.Model) []Model {
	td := r.ensureTenant(t)
	td.lock.RLock()
	defer td.lock.RUnlock()

	results := make([]Model, 0, len(td.instances))
	for _, m := range td.instances {
		results = append(results, m)
	}
	return results
}

func (r *Registry) Update(t tenant.Model, instanceId uuid.UUID, updaters ...func(m Model) Model) (Model, error) {
	td := r.ensureTenant(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	oldModel, ok := td.instances[instanceId]
	if !ok {
		return Model{}, ErrNotFound
	}

	newModel := oldModel
	for _, updater := range updaters {
		newModel = updater(newModel)
	}

	// Update character index
	updateCharacterIndex(td, oldModel, newModel)

	td.instances[instanceId] = newModel
	return newModel, nil
}

func (r *Registry) Remove(t tenant.Model, instanceId uuid.UUID) {
	td := r.ensureTenant(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	if m, ok := td.instances[instanceId]; ok {
		for _, c := range m.Characters() {
			delete(td.characterToInstance, c.CharacterId())
		}
		delete(td.instances, instanceId)
	}
}

func (r *Registry) Clear(t tenant.Model) {
	td := r.ensureTenant(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	td.instances = make(map[uuid.UUID]Model)
	td.characterToInstance = make(map[uint32]uuid.UUID)
}

func (r *Registry) ResetForTesting() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.tenants = make(map[tenant.Model]*tenantData)
}

func updateCharacterIndex(td *tenantData, oldModel, newModel Model) {
	oldChars := make(map[uint32]bool)
	for _, c := range oldModel.Characters() {
		oldChars[c.CharacterId()] = true
	}

	newChars := make(map[uint32]bool)
	for _, c := range newModel.Characters() {
		newChars[c.CharacterId()] = true
	}

	for cid := range oldChars {
		if !newChars[cid] {
			delete(td.characterToInstance, cid)
		}
	}

	for cid := range newChars {
		if !oldChars[cid] {
			td.characterToInstance[cid] = newModel.Id()
		}
	}
}
