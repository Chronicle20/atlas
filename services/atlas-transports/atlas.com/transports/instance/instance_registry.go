package instance

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type RouteKey struct {
	TenantId uuid.UUID
	RouteId  uuid.UUID
}

type InstanceRegistry struct {
	mu        sync.RWMutex
	instances map[uuid.UUID]*TransportInstance
	byRoute   map[RouteKey][]*TransportInstance
}

var instanceRegistry *InstanceRegistry
var instanceRegistryOnce sync.Once

func getInstanceRegistry() *InstanceRegistry {
	instanceRegistryOnce.Do(func() {
		instanceRegistry = &InstanceRegistry{
			instances: make(map[uuid.UUID]*TransportInstance),
			byRoute:   make(map[RouteKey][]*TransportInstance),
		}
	})
	return instanceRegistry
}

// FindOrCreateInstance finds an existing boarding instance with room and an open window,
// or creates a new one with a fresh UUID.
func (r *InstanceRegistry) FindOrCreateInstance(tenantId uuid.UUID, route RouteModel, now time.Time) *TransportInstance {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := RouteKey{TenantId: tenantId, RouteId: route.Id()}

	// Look for existing instance in Boarding state with room and open window
	if instances, ok := r.byRoute[key]; ok {
		for _, inst := range instances {
			if inst.state == Boarding && uint32(inst.CharacterCount()) < route.Capacity() && now.Before(inst.boardingUntil) {
				return inst
			}
		}
	}

	// Create new instance
	instanceId := uuid.New()
	boardingUntil := now.Add(route.BoardingWindow())
	arrivalAt := boardingUntil.Add(route.TravelDuration())
	inst := NewTransportInstance(instanceId, route.Id(), tenantId, boardingUntil, arrivalAt)
	instPtr := &inst
	r.instances[instanceId] = instPtr
	r.byRoute[key] = append(r.byRoute[key], instPtr)
	return instPtr
}

// AddCharacter adds a character to an instance.
func (r *InstanceRegistry) AddCharacter(instanceId uuid.UUID, entry CharacterEntry) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.instances[instanceId]
	if !ok {
		return false
	}
	inst.characters = append(inst.characters, entry)
	return true
}

// RemoveCharacter removes a character from an instance.
// Returns true if the instance is now empty.
func (r *InstanceRegistry) RemoveCharacter(instanceId uuid.UUID, characterId uint32) (empty bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.instances[instanceId]
	if !ok {
		return false
	}

	for i, c := range inst.characters {
		if c.CharacterId == characterId {
			inst.characters = append(inst.characters[:i], inst.characters[i+1:]...)
			break
		}
	}
	return len(inst.characters) == 0
}

// TransitionToInTransit transitions an instance from Boarding to InTransit.
func (r *InstanceRegistry) TransitionToInTransit(instanceId uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.instances[instanceId]
	if !ok || inst.state != Boarding {
		return false
	}
	inst.state = InTransit
	return true
}

// ReleaseInstance removes an instance from all maps.
func (r *InstanceRegistry) ReleaseInstance(instanceId uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.instances[instanceId]
	if !ok {
		return
	}

	key := RouteKey{TenantId: inst.tenantId, RouteId: inst.routeId}
	if instances, ok := r.byRoute[key]; ok {
		for i, existing := range instances {
			if existing.instanceId == instanceId {
				r.byRoute[key] = append(instances[:i], instances[i+1:]...)
				break
			}
		}
		if len(r.byRoute[key]) == 0 {
			delete(r.byRoute, key)
		}
	}
	delete(r.instances, instanceId)
}

// GetInstance returns the instance for a given instance ID.
func (r *InstanceRegistry) GetInstance(instanceId uuid.UUID) (*TransportInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.instances[instanceId]
	return inst, ok
}

// GetExpiredBoarding returns instances past their boardingUntil still in Boarding state.
func (r *InstanceRegistry) GetExpiredBoarding(now time.Time) []*TransportInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*TransportInstance
	for _, inst := range r.instances {
		if inst.state == Boarding && now.After(inst.boardingUntil) {
			result = append(result, inst)
		}
	}
	return result
}

// GetExpiredTransit returns instances past their arrivalAt.
func (r *InstanceRegistry) GetExpiredTransit(now time.Time) []*TransportInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*TransportInstance
	for _, inst := range r.instances {
		if inst.state == InTransit && now.After(inst.arrivalAt) {
			result = append(result, inst)
		}
	}
	return result
}

// GetAllActive returns all active instances.
func (r *InstanceRegistry) GetAllActive() []*TransportInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*TransportInstance
	for _, inst := range r.instances {
		result = append(result, inst)
	}
	return result
}

// GetStuck returns instances exceeding the given max lifetime.
func (r *InstanceRegistry) GetStuck(now time.Time, maxLifetime time.Duration) []*TransportInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*TransportInstance
	for _, inst := range r.instances {
		if now.Sub(inst.createdAt) > maxLifetime {
			result = append(result, inst)
		}
	}
	return result
}
