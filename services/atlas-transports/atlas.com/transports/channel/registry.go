package channel

import (
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"

	"github.com/google/uuid"
)

type Registry struct {
	mu    sync.RWMutex
	store map[uuid.UUID][]channel.Model
}

var (
	instance *Registry
	once     sync.Once
)

// GetRegistry returns the global singleton instance
func getRegistry() *Registry {
	once.Do(func() {
		instance = &Registry{
			store: make(map[uuid.UUID][]channel.Model),
		}
	})
	return instance
}

// Add adds a model to the given tenant's list
func (r *Registry) Add(tenantId uuid.UUID, model channel.Model) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if the model already exists for this tenant
	models, ok := r.store[tenantId]
	if ok {
		for _, m := range models {
			// If a model with the same worldId and id already exists, don't add it again
			if m.WorldId() == model.WorldId() && m.Id() == model.Id() {
				return
			}
		}
	}

	r.store[tenantId] = append(r.store[tenantId], model)
}

// Remove removes a model by its ID from the given tenant's list
func (r *Registry) Remove(tenantId uuid.UUID, ch channel.Model) {
	r.mu.Lock()
	defer r.mu.Unlock()

	models, ok := r.store[tenantId]
	if !ok {
		return
	}

	filtered := models[:0]
	for _, m := range models {
		if m.WorldId() != ch.WorldId() || m.Id() != ch.Id() {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		delete(r.store, tenantId)
	} else {
		r.store[tenantId] = filtered
	}
}

// GetAll returns a copy of all models for a given tenant
func (r *Registry) GetAll(tenantId uuid.UUID) []channel.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models, ok := r.store[tenantId]
	if !ok {
		return nil
	}

	// Return a copy to prevent external modification
	copyModels := make([]channel.Model, len(models))
	copy(copyModels, models)
	return copyModels
}
