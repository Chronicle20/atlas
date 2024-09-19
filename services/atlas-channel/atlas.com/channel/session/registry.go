package session

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	mutex           sync.RWMutex
	sessionRegistry map[uuid.UUID]map[uuid.UUID]Model
	lockRegistry    map[uuid.UUID]map[uuid.UUID]*sync.RWMutex
}

var sessionRegistryOnce sync.Once
var sessionRegistry *Registry

func GetRegistry() *Registry {
	sessionRegistryOnce.Do(func() {
		sessionRegistry = &Registry{}
		sessionRegistry.sessionRegistry = make(map[uuid.UUID]map[uuid.UUID]Model)
		sessionRegistry.lockRegistry = make(map[uuid.UUID]map[uuid.UUID]*sync.RWMutex)
	})
	return sessionRegistry
}

func (r *Registry) Add(s Model) {
	r.mutex.Lock()
	t := s.Tenant()
	if _, ok := r.sessionRegistry[t.Id()]; !ok {
		r.sessionRegistry[t.Id()] = make(map[uuid.UUID]Model)
	}
	r.sessionRegistry[t.Id()][s.SessionId()] = s

	if _, ok := r.lockRegistry[t.Id()]; !ok {
		r.lockRegistry[t.Id()] = make(map[uuid.UUID]*sync.RWMutex)
	}
	r.lockRegistry[t.Id()][s.SessionId()] = &sync.RWMutex{}
	r.mutex.Unlock()
}

func (r *Registry) Remove(tenantId uuid.UUID, sessionId uuid.UUID) {
	r.mutex.Lock()
	delete(r.sessionRegistry[tenantId], sessionId)
	delete(r.lockRegistry[tenantId], sessionId)
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

func (r *Registry) GetLock(tenant tenant.Model, sessionId uuid.UUID) (*sync.RWMutex, bool) {
	r.mutex.RLock()
	if _, ok := r.lockRegistry[tenant.Id()]; !ok {
		r.mutex.RUnlock()
		return nil, false
	}

	if val, ok := r.lockRegistry[tenant.Id()][sessionId]; ok {
		r.mutex.RUnlock()
		return val, true
	}
	r.mutex.RUnlock()
	return nil, false
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

func (r *Registry) Update(m Model) {
	r.mutex.Lock()
	t := m.Tenant()
	if _, ok := r.sessionRegistry[t.Id()]; !ok {
		r.sessionRegistry[t.Id()] = make(map[uuid.UUID]Model)
	}
	r.sessionRegistry[t.Id()][m.SessionId()] = m
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
