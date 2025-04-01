package server

import (
	"sync"
)

var registry *Registry
var once sync.Once

type Registry struct {
	lock     sync.RWMutex
	registry []Model
}

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{
			registry: make([]Model, 0),
		}
	})
	return registry
}

func (r *Registry) Register(m Model) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.registry = append(r.registry, m)
}

func (r *Registry) GetAll() []Model {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.registry
}
