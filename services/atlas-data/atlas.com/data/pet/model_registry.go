package pet

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type PetModelRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var mmReg *PetModelRegistry
var mmOnce sync.Once

func GetPetModelRegistry() *PetModelRegistry {
	mmOnce.Do(func() {
		mmReg = &PetModelRegistry{}
		mmReg.registry = make(map[tenant.Model]map[uint32]Model)
		mmReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return mmReg
}

func (r *PetModelRegistry) ensureTenantLock(t tenant.Model) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.tenantLock[t]; !ok {
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]Model)
	}
}

func (r *PetModelRegistry) Add(t tenant.Model, m Model) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]Model)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	r.registry[t][m.Id()] = m
	return nil
}

func (r *PetModelRegistry) Get(t tenant.Model, petId uint32) (Model, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	if val, ok := r.registry[t][petId]; ok {
		return val, nil
	}
	return Model{}, errors.New("not found")
}

func (r *PetModelRegistry) GetAll(t tenant.Model) ([]Model, error) {
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
