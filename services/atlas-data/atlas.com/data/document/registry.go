package document

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Registry[I uint32, M Identifier[I]] struct {
	lock       sync.Mutex
	registry   map[tenant.Model]map[I]M
	tenantLock map[tenant.Model]*sync.RWMutex
}

func NewRegistry[I uint32, M Identifier[I]]() *Registry[I, M] {
	return &Registry[I, M]{
		registry:   make(map[tenant.Model]map[I]M),
		tenantLock: make(map[tenant.Model]*sync.RWMutex),
	}
}

func (r *Registry[I, M]) ensureTenantLock(t tenant.Model) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.tenantLock[t]; !ok {
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[I]M)
	}
}

func (r *Registry[I, M]) Add(t tenant.Model, m M) (M, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	r.registry[t][m.GetId()] = m
	return m, nil
}

func (r *Registry[I, M]) Get(t tenant.Model, id I) (M, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	var val M
	var ok bool
	if val, ok = r.registry[t][id]; ok {
		return val, nil
	}
	return val, errors.New("not found")
}

func (r *Registry[I, M]) GetAll(t tenant.Model) ([]M, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	var res = make([]M, 0)
	var tr map[I]M
	var ok bool
	if tr, ok = r.registry[t]; !ok || len(tr) == 0 {
		return res, errors.New("not found")
	}

	for _, m := range tr {
		res = append(res, m)
	}
	return res, nil
}

func (r *Registry[I, M]) Clear(t tenant.Model) error {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.registry, t)
	r.registry[t] = make(map[I]M)
	return nil
}

func (r *Registry[I, M]) Count() int {
	return len(r.registry)
}
