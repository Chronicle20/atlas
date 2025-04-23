package session

import (
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	mutex           sync.RWMutex
	sessionRegistry map[uuid.UUID]map[uuid.UUID]Model
}

var sessionRegistryOnce sync.Once
var sessionRegistry *Registry

func getRegistry() *Registry {
	sessionRegistryOnce.Do(func() {
		sessionRegistry = &Registry{}
		sessionRegistry.sessionRegistry = make(map[uuid.UUID]map[uuid.UUID]Model)
	})
	return sessionRegistry
}

func (r *Registry) Add(tenantId uuid.UUID, s Model) {
	r.mutex.Lock()
	if _, ok := r.sessionRegistry[tenantId]; !ok {
		r.sessionRegistry[tenantId] = make(map[uuid.UUID]Model)
	}
	r.sessionRegistry[tenantId][s.SessionId()] = s
	r.mutex.Unlock()
}

func (r *Registry) Remove(tenantId uuid.UUID, sessionId uuid.UUID) {
	r.mutex.Lock()
	delete(r.sessionRegistry[tenantId], sessionId)
	r.mutex.Unlock()
}

func (r *Registry) Get(tenantId uuid.UUID, sessionId uuid.UUID) (Model, bool) {
	r.mutex.RLock()
	if _, ok := r.sessionRegistry[tenantId]; !ok {
		r.mutex.RUnlock()
		return Model{}, false
	}

	if s, ok := r.sessionRegistry[tenantId][sessionId]; ok {
		r.mutex.RUnlock()
		return s, true
	}
	r.mutex.RUnlock()
	return Model{}, false
}

func (r *Registry) GetAll() []Model {
	r.mutex.RLock()
	s := make([]Model, 0)
	for _, rs := range r.sessionRegistry {
		for _, v := range rs {
			s = append(s, v)
		}
	}
	r.mutex.RUnlock()
	return s
}

func (r *Registry) Update(tenantId uuid.UUID, m Model) {
	r.mutex.Lock()
	if _, ok := r.sessionRegistry[tenantId]; !ok {
		r.sessionRegistry[tenantId] = make(map[uuid.UUID]Model)
	}
	r.sessionRegistry[tenantId][m.SessionId()] = m
	r.mutex.Unlock()
}

func (r *Registry) GetInTenant(id uuid.UUID) []Model {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	s := make([]Model, 0)
	if _, ok := r.sessionRegistry[id]; !ok {
		return s
	}

	for _, v := range r.sessionRegistry[id] {
		s = append(s, v)
	}
	return s
}
