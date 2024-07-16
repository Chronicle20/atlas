package tenant

import (
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	mutex   sync.RWMutex
	tenants map[uuid.UUID]bool
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenants = make(map[uuid.UUID]bool)
	})
	return registry
}

func (r *Registry) Add(id uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tenants[id] = true
}

func (r *Registry) Remove(id uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.tenants, id)
}

func (r *Registry) Contains(id uuid.UUID) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.tenants[id]
}

func (r *Registry) GetAll() []uuid.UUID {
	r.mutex.RLock()
	r.mutex.RUnlock()
	var keys []uuid.UUID
	for k := range r.tenants {
		keys = append(keys, k)
	}
	return keys
}
