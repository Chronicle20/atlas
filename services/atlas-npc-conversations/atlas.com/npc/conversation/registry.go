package conversation

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Registry struct {
	lock       sync.RWMutex
	registry   map[tenant.Model]map[uint32]ConversationContext
	tenantLock map[tenant.Model]*sync.RWMutex
}

var once sync.Once
var registry *Registry

func GetRegistry() *Registry {
	once.Do(func() {
		registry = initRegistry()
	})
	return registry
}

func initRegistry() *Registry {
	s := &Registry{
		lock:       sync.RWMutex{},
		registry:   make(map[tenant.Model]map[uint32]ConversationContext),
		tenantLock: make(map[tenant.Model]*sync.RWMutex),
	}
	return s
}

func (s *Registry) GetPreviousContext(t tenant.Model, characterId uint32) (ConversationContext, error) {
	s.lock.Lock()
	if _, ok := s.registry[t]; !ok {
		s.registry[t] = make(map[uint32]ConversationContext)
		s.tenantLock[t] = &sync.RWMutex{}
	}
	tl := s.tenantLock[t]
	s.lock.Unlock()

	tl.RLock()
	if val, ok := s.registry[t][characterId]; ok {
		tl.RUnlock()
		return val, nil
	}
	tl.RUnlock()
	return ConversationContext{}, errors.New("unable to previous context")
}

func (s *Registry) SetContext(t tenant.Model, characterId uint32, ctx ConversationContext) {
	s.lock.Lock()
	if _, ok := s.registry[t]; !ok {
		s.registry[t] = make(map[uint32]ConversationContext)
		s.tenantLock[t] = &sync.RWMutex{}
	}
	tl := s.tenantLock[t]
	s.lock.Unlock()

	tl.Lock()
	s.registry[t][characterId] = ctx
	tl.Unlock()
}

func (s *Registry) ClearContext(t tenant.Model, characterId uint32) {
	s.lock.Lock()
	if _, ok := s.registry[t]; !ok {
		s.registry[t] = make(map[uint32]ConversationContext)
		s.tenantLock[t] = &sync.RWMutex{}
	}
	tl := s.tenantLock[t]
	s.lock.Unlock()

	tl.Lock()
	delete(s.registry[t], characterId)
	tl.Unlock()
}
