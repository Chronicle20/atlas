package session

import (
	"sync"

	"github.com/google/uuid"
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
	defer r.mutex.Unlock()

	if _, ok := r.sessionRegistry[tenantId]; !ok {
		r.sessionRegistry[tenantId] = make(map[uuid.UUID]Model)
	}
	r.sessionRegistry[tenantId][s.SessionId()] = s
}

func (r *Registry) Remove(tenantId uuid.UUID, sessionId uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.sessionRegistry[tenantId], sessionId)
}

func (r *Registry) Get(tenantId uuid.UUID, sessionId uuid.UUID) (Model, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if _, ok := r.sessionRegistry[tenantId]; !ok {
		return Model{}, false
	}

	if s, ok := r.sessionRegistry[tenantId][sessionId]; ok {
		return s, true
	}
	return Model{}, false
}

func (r *Registry) Update(tenantId uuid.UUID, m Model) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.sessionRegistry[tenantId]; !ok {
		r.sessionRegistry[tenantId] = make(map[uuid.UUID]Model)
	}
	r.sessionRegistry[tenantId][m.SessionId()] = m
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
