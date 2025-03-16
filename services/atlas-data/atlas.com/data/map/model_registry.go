package _map

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type MapModelRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var mmReg *MapModelRegistry
var mmOnce sync.Once

func GetMapModelRegistry() *MapModelRegistry {
	mmOnce.Do(func() {
		mmReg = &MapModelRegistry{}
		mmReg.registry = make(map[tenant.Model]map[uint32]Model)
		mmReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return mmReg
}

func (r *MapModelRegistry) ensureTenantLock(t tenant.Model) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.tenantLock[t]; !ok {
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]Model)
	}
}

func (r *MapModelRegistry) Add(t tenant.Model, m Model) error {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	r.registry[t][m.Id()] = m
	return nil
}

func (r *MapModelRegistry) Get(t tenant.Model, mapId uint32) (Model, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	if val, ok := r.registry[t][mapId]; ok {
		return val, nil
	}
	return Model{}, errors.New("not found")
}

func (r *MapModelRegistry) GetAll(t tenant.Model) ([]Model, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	var res = make([]Model, 0)
	var tr map[uint32]Model
	var ok bool
	if tr, ok = r.registry[t]; !ok || len(tr) == 0 {
		return res, errors.New("not found")
	}

	for _, m := range tr {
		res = append(res, m)
	}
	return res, nil
}
