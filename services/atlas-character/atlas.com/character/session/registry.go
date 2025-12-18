package session

import (
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
	"time"
)

type Registry struct {
	mutex           sync.RWMutex
	sessionRegistry map[uuid.UUID]map[uint32]Model
	lockRegistry    map[uuid.UUID]*sync.RWMutex
}

var once sync.Once
var registry *Registry

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.sessionRegistry = make(map[uuid.UUID]map[uint32]Model)
		registry.lockRegistry = make(map[uuid.UUID]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) Add(t tenant.Model, characterId uint32, worldId world.Id, channelId channel.Id, state State) error {
	r.mutex.Lock()
	if _, ok := r.lockRegistry[t.Id()]; !ok {
		r.lockRegistry[t.Id()] = &sync.RWMutex{}
		r.sessionRegistry[t.Id()] = make(map[uint32]Model)
	}
	r.mutex.Unlock()

	r.lockRegistry[t.Id()].Lock()
	defer r.lockRegistry[t.Id()].Unlock()
	if val, ok := r.sessionRegistry[t.Id()][characterId]; ok {
		if val.State() == StateLoggedIn {
			return errors.New("already logged in")
		}
	}

	r.sessionRegistry[t.Id()][characterId] = Model{
		tenant:      t,
		characterId: characterId,
		worldId:     worldId,
		channelId:   channelId,
		state:       state,
		age:         time.Now(),
	}
	return nil
}

func (r *Registry) Set(t tenant.Model, characterId uint32, worldId world.Id, channelId channel.Id, state State) error {
	r.mutex.Lock()
	if _, ok := r.lockRegistry[t.Id()]; !ok {
		r.lockRegistry[t.Id()] = &sync.RWMutex{}
		r.sessionRegistry[t.Id()] = make(map[uint32]Model)
	}
	r.mutex.Unlock()

	r.lockRegistry[t.Id()].Lock()
	defer r.lockRegistry[t.Id()].Unlock()

	r.sessionRegistry[t.Id()][characterId] = Model{
		tenant:      t,
		characterId: characterId,
		worldId:     worldId,
		channelId:   channelId,
		state:       state,
		age:         time.Now(),
	}
	return nil
}

func (r *Registry) Get(t tenant.Model, characterId uint32) (Model, error) {
	if _, ok := r.lockRegistry[t.Id()]; !ok {
		r.mutex.Lock()
		r.lockRegistry[t.Id()] = &sync.RWMutex{}
		r.sessionRegistry[t.Id()] = make(map[uint32]Model)
		r.mutex.Unlock()
	}

	r.lockRegistry[t.Id()].RLock()
	defer r.lockRegistry[t.Id()].RUnlock()
	if val, ok := r.sessionRegistry[t.Id()][characterId]; ok {
		return val, nil
	}
	return Model{}, errors.New("not found")
}

func (r *Registry) GetAll() []Model {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var results = make([]Model, 0)

	for t, cm := range r.sessionRegistry {
		r.lockRegistry[t].RLock()
		for _, c := range cm {
			results = append(results, c)
		}
		r.lockRegistry[t].RUnlock()
	}

	return results
}

func (r *Registry) Remove(t tenant.Model, characterId uint32) {
	if _, ok := r.lockRegistry[t.Id()]; !ok {
		r.mutex.Lock()
		r.lockRegistry[t.Id()] = &sync.RWMutex{}
		r.sessionRegistry[t.Id()] = make(map[uint32]Model)
		r.mutex.Unlock()
	}

	r.lockRegistry[t.Id()].Lock()
	defer r.lockRegistry[t.Id()].Unlock()
	delete(r.sessionRegistry[t.Id()], characterId)
}
