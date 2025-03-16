package consumable

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type ConsumableModelRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var mmReg *ConsumableModelRegistry
var mmOnce sync.Once

func GetConsumableModelRegistry() *ConsumableModelRegistry {
	mmOnce.Do(func() {
		mmReg = &ConsumableModelRegistry{}
		mmReg.registry = make(map[tenant.Model]map[uint32]Model)
		mmReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return mmReg
}

func (r *ConsumableModelRegistry) ensureTenantLock(t tenant.Model) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.tenantLock[t]; !ok {
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]Model)
	}
}

func (r *ConsumableModelRegistry) Add(t tenant.Model, m Model) error {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	r.registry[t][m.Id()] = m
	return nil
}

func (r *ConsumableModelRegistry) Get(t tenant.Model, consumableId uint32) (Model, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	if val, ok := r.registry[t][consumableId]; ok {
		return val, nil
	}
	return Model{}, errors.New("not found")
}

func (r *ConsumableModelRegistry) GetAll(t tenant.Model) ([]Model, error) {
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
