package expression

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
)

type Registry struct {
	lock          sync.Mutex
	expressionReg map[tenant.Model]map[uint32]Model
	tenantLock    map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.expressionReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

// getOrCreateTenantMaps returns the expression map and lock for a tenant,
// creating them if they don't exist. This method is thread-safe.
func (r *Registry) getOrCreateTenantMaps(t tenant.Model) (map[uint32]Model, *sync.RWMutex) {
	r.lock.Lock()
	defer r.lock.Unlock()

	em, ok := r.expressionReg[t]
	if !ok {
		em = make(map[uint32]Model)
		r.expressionReg[t] = em
	}

	tl, ok := r.tenantLock[t]
	if !ok {
		tl = &sync.RWMutex{}
		r.tenantLock[t] = tl
	}

	return em, tl
}

func (r *Registry) add(t tenant.Model, characterId uint32, field field.Model, expression uint32) Model {
	em, tl := r.getOrCreateTenantMaps(t)

	tl.Lock()
	defer tl.Unlock()

	expiration := time.Now().Add(time.Second * time.Duration(5))

	e := NewModelBuilder(t).
		SetCharacterId(characterId).
		SetLocation(field).
		SetExpression(expression).
		SetExpiration(expiration).
		MustBuild()

	em[characterId] = e
	return e
}

func (r *Registry) popExpired() []Model {
	var results = make([]Model, 0)
	now := time.Now()
	r.lock.Lock()
	defer r.lock.Unlock()
	for t, cm := range r.expressionReg {
		r.tenantLock[t].Lock()
		for id, m := range cm {
			if now.Sub(m.Expiration()) > 0 {
				results = append(results, m)
				delete(r.expressionReg[t], id)
			}
		}
		r.tenantLock[t].Unlock()
	}
	return results
}

func (r *Registry) clear(t tenant.Model, characterId uint32) {
	em, tl := r.getOrCreateTenantMaps(t)

	tl.Lock()
	defer tl.Unlock()
	delete(em, characterId)
}

// get retrieves an expression for a character. Returns the model and true if found.
func (r *Registry) get(t tenant.Model, characterId uint32) (Model, bool) {
	em, tl := r.getOrCreateTenantMaps(t)

	tl.RLock()
	defer tl.RUnlock()
	if m, ok := em[characterId]; ok {
		return m, true
	}
	return Model{}, false
}

// ResetForTesting clears all registry state. Only for use in tests.
func (r *Registry) ResetForTesting() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.expressionReg = make(map[tenant.Model]map[uint32]Model)
	r.tenantLock = make(map[tenant.Model]*sync.RWMutex)
}
