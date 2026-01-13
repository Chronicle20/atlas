package invite

import (
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
)

type Registry struct {
	lock           sync.RWMutex
	tenantInviteId map[tenant.Model]uint32
	inviteReg      map[tenant.Model]map[uint32]map[string][]Model
	tenantLock     map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenantInviteId = make(map[tenant.Model]uint32)
		registry.inviteReg = make(map[tenant.Model]map[uint32]map[string][]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

// getOrCreateTenantLock safely retrieves or creates the per-tenant lock.
// This method ensures thread-safe access to the tenantLock map.
func (r *Registry) getOrCreateTenantLock(t tenant.Model) *sync.RWMutex {
	// First try with read lock (fast path for existing tenants)
	r.lock.RLock()
	if tl, ok := r.tenantLock[t]; ok {
		r.lock.RUnlock()
		return tl
	}
	r.lock.RUnlock()

	// Upgrade to write lock to create new tenant structures
	r.lock.Lock()
	defer r.lock.Unlock()

	// Double-check after acquiring write lock (another goroutine may have created it)
	if tl, ok := r.tenantLock[t]; ok {
		return tl
	}

	// Create new tenant structures
	tl := &sync.RWMutex{}
	r.inviteReg[t] = make(map[uint32]map[string][]Model)
	r.tenantLock[t] = tl
	return tl
}

func (r *Registry) Create(t tenant.Model, originatorId uint32, worldId byte, targetId uint32, inviteType string, referenceId uint32) Model {
	var inviteId uint32

	// Get next invite ID while holding write lock
	r.lock.Lock()
	if id, ok := r.tenantInviteId[t]; ok {
		inviteId = id + 1
	} else {
		inviteId = StartInviteId
		// Initialize tenant structures if not exists
		if _, ok := r.tenantLock[t]; !ok {
			r.inviteReg[t] = make(map[uint32]map[string][]Model)
			r.tenantLock[t] = &sync.RWMutex{}
		}
	}
	r.tenantInviteId[t] = inviteId
	tenantLock := r.tenantLock[t]
	r.lock.Unlock()

	m, err := NewBuilder().
		SetTenant(t).
		SetId(inviteId).
		SetInviteType(inviteType).
		SetReferenceId(referenceId).
		SetOriginatorId(originatorId).
		SetTargetId(targetId).
		SetWorldId(worldId).
		SetAge(time.Now()).
		Build()
	if err != nil {
		// This should never happen as the registry generates valid IDs
		// and callers should provide valid parameters
		panic("invite.Registry.Create: builder validation failed: " + err.Error())
	}

	tenantLock.Lock()
	defer tenantLock.Unlock()

	// Access tenant registry safely (we have the tenant lock)
	r.lock.RLock()
	tenReg := r.inviteReg[t]
	r.lock.RUnlock()

	if _, ok := tenReg[targetId]; !ok {
		tenReg[targetId] = make(map[string][]Model)
	}

	if _, ok := tenReg[targetId][inviteType]; !ok {
		tenReg[targetId][inviteType] = make([]Model, 0)
	}

	for _, i := range tenReg[targetId][inviteType] {
		if i.ReferenceId() == referenceId {
			return i
		}
	}
	tenReg[targetId][inviteType] = append(tenReg[targetId][inviteType], m)
	return m
}

func (r *Registry) GetByOriginator(t tenant.Model, actorId uint32, inviteType string, originatorId uint32) (Model, error) {
	tl := r.getOrCreateTenantLock(t)

	tl.RLock()
	defer tl.RUnlock()

	r.lock.RLock()
	tenReg := r.inviteReg[t]
	r.lock.RUnlock()

	if charReg, ok := tenReg[actorId]; ok {
		if invReg, ok := charReg[inviteType]; ok {
			for _, i := range invReg {
				if i.OriginatorId() == originatorId {
					return i, nil
				}
			}
		}
	}
	return Model{}, errors.New("not found")
}

func (r *Registry) GetByReference(t tenant.Model, actorId uint32, inviteType string, referenceId uint32) (Model, error) {
	tl := r.getOrCreateTenantLock(t)

	tl.RLock()
	defer tl.RUnlock()

	r.lock.RLock()
	tenReg := r.inviteReg[t]
	r.lock.RUnlock()

	if charReg, ok := tenReg[actorId]; ok {
		if invReg, ok := charReg[inviteType]; ok {
			for _, i := range invReg {
				if i.ReferenceId() == referenceId {
					return i, nil
				}
			}
		}
	}
	return Model{}, errors.New("not found")
}

func (r *Registry) GetForCharacter(t tenant.Model, characterId uint32) ([]Model, error) {
	tl := r.getOrCreateTenantLock(t)

	tl.RLock()
	defer tl.RUnlock()

	r.lock.RLock()
	tenReg := r.inviteReg[t]
	r.lock.RUnlock()

	var results = make([]Model, 0)
	if charReg, ok := tenReg[characterId]; ok {
		for _, v := range charReg {
			results = append(results, v...)
		}
	}
	return results, nil
}

func (r *Registry) Delete(t tenant.Model, actorId uint32, inviteType string, originatorId uint32) error {
	tl := r.getOrCreateTenantLock(t)

	tl.Lock()
	defer tl.Unlock()

	r.lock.RLock()
	tenReg := r.inviteReg[t]
	r.lock.RUnlock()

	if charReg, ok := tenReg[actorId]; ok {
		if invReg, ok := charReg[inviteType]; ok {
			var found = false
			var remain = make([]Model, 0)
			for _, i := range invReg {
				if i.OriginatorId() != originatorId {
					remain = append(remain, i)
				} else {
					found = true
				}
			}
			tenReg[actorId][inviteType] = remain
			if found {
				return nil
			}
		}
	}
	return errors.New("not found")
}

func (r *Registry) GetExpired(timeout time.Duration) ([]Model, error) {
	var results = make([]Model, 0)

	// Get a snapshot of tenant keys while holding read lock
	r.lock.RLock()
	tenants := make([]tenant.Model, 0, len(r.inviteReg))
	for k := range r.inviteReg {
		tenants = append(tenants, k)
	}
	r.lock.RUnlock()

	// Process each tenant
	for _, t := range tenants {
		r.lock.RLock()
		tl, ok := r.tenantLock[t]
		tenReg := r.inviteReg[t]
		r.lock.RUnlock()

		if !ok || tenReg == nil {
			continue
		}

		tl.RLock()
		for _, cir := range tenReg {
			for _, is := range cir {
				for _, i := range is {
					if i.Expired(timeout) {
						results = append(results, i)
					}
				}
			}
		}
		tl.RUnlock()
	}
	return results, nil
}
