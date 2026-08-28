package backeffect

import (
	"sync"
)

type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey][]BackEffectEntry
}

var (
	registry *Registry
	once     sync.Once
)

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.entries = make(map[FieldKey][]BackEffectEntry)
	})
	return registry
}
