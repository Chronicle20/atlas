package action

import (
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// PendingAction represents a pending portal action awaiting saga completion
type PendingAction struct {
	CharacterId    uint32
	WorldId        world.Id
	ChannelId      channel.Id
	FailureMessage string // Message to display on failure
}

// Registry tracks pending portal actions by saga ID
type Registry struct {
	lock sync.RWMutex
	// tenantId → sagaId → pending action
	registry map[uuid.UUID]map[uuid.UUID]PendingAction
}

var registryOnce sync.Once
var registryInstance *Registry

// GetRegistry returns the singleton registry instance
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registryInstance = &Registry{
			registry: make(map[uuid.UUID]map[uuid.UUID]PendingAction),
		}
	})
	return registryInstance
}

// Add registers a pending action for a saga
func (r *Registry) Add(tenantId, sagaId uuid.UUID, action PendingAction) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.registry[tenantId]; !ok {
		r.registry[tenantId] = make(map[uuid.UUID]PendingAction)
	}
	r.registry[tenantId][sagaId] = action
}

// Get retrieves a pending action by saga ID
func (r *Registry) Get(tenantId, sagaId uuid.UUID) (PendingAction, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if tenantRegistry, ok := r.registry[tenantId]; ok {
		if action, ok := tenantRegistry[sagaId]; ok {
			return action, true
		}
	}
	return PendingAction{}, false
}

// Remove removes a pending action by saga ID
func (r *Registry) Remove(tenantId, sagaId uuid.UUID) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if tenantRegistry, ok := r.registry[tenantId]; ok {
		delete(tenantRegistry, sagaId)
	}
}
