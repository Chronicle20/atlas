package tenant

import (
	"sync"

	"github.com/google/uuid"
)

type Registry struct {
	mutex   sync.RWMutex
	tenants map[uuid.UUID]Model
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenants = make(map[uuid.UUID]Model)
	})
	return registry
}

func (r *Registry) Add(tenant Model) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tenants[tenant.Id()] = tenant
}

func (r *Registry) Remove(id uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.tenants, id)
}

func (r *Registry) Contains(id uuid.UUID) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if _, ok := r.tenants[id]; ok {
		return true
	}
	return false
}

func (r *Registry) GetAll() []Model {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var values []Model
	for _, v := range r.tenants {
		values = append(values, v)
	}
	return values
}
