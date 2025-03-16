package registry

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Identifier[I comparable] interface {
	Id() I
}
type Registry[I comparable, M Identifier[I]] struct {
	lock       sync.Mutex
	registry   map[tenant.Model]map[I]M
	tenantLock map[tenant.Model]*sync.RWMutex
}

func NewRegistry[I comparable, M Identifier[I]]() *Registry[I, M] {
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

func (r *Registry[I, M]) Add(t tenant.Model, m M) error {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	r.registry[t][m.Id()] = m
	return nil
}

func (r *Registry[I, M]) Get(t tenant.Model, consumableId I) (M, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	var val M
	var ok bool
	if val, ok = r.registry[t][consumableId]; ok {
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
