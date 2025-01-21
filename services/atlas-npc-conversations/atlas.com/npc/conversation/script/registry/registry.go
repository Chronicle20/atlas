package registry

import (
	"atlas-npc-conversations/conversation/script"
	"atlas-npc-conversations/conversation/script/discrete"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	registry map[uuid.UUID]map[uint32]script.Script
	scripts  map[string]script.Script
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
		registry: make(map[uuid.UUID]map[uint32]script.Script),
		scripts:  make(map[string]script.Script),
	}
	s.addConversation(discrete.Heracle{})
	return s
}

func (s *Registry) GetScript(t tenant.Model, npcId uint32) (*script.Script, error) {
	var ok bool
	if _, ok = s.registry[t.Id()]; !ok {
		return nil, errors.New("tenant not configured")
	}

	var val script.Script
	if val, ok = s.registry[t.Id()][npcId]; !ok {
		return nil, errors.New("unable to locate script")
	}
	return &val, nil
}

func (s *Registry) addConversation(handler script.Script) {
	s.scripts[handler.Name()] = handler
}

func (s *Registry) InitScript(tenantId uuid.UUID, npcId uint32, impl string) {
	if _, ok := s.registry[tenantId]; !ok {
		s.registry[tenantId] = make(map[uint32]script.Script)
	}
	if f, ok := s.scripts[impl]; ok {
		s.registry[tenantId][npcId] = f
	}
}
