package equipment

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type EquipmentModelRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var mmReg *EquipmentModelRegistry
var mmOnce sync.Once

func GetEquipmentModelRegistry() *EquipmentModelRegistry {
	mmOnce.Do(func() {
		mmReg = &EquipmentModelRegistry{}
		mmReg.registry = make(map[tenant.Model]map[uint32]Model)
		mmReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return mmReg
}

func (r *EquipmentModelRegistry) ensureTenantLock(t tenant.Model) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.tenantLock[t]; !ok {
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]Model)
	}
}

func (r *EquipmentModelRegistry) Add(t tenant.Model, m Model) error {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	r.registry[t][m.Id()] = m
	return nil
}

func (r *EquipmentModelRegistry) Get(t tenant.Model, equipmentId uint32) (Model, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	if val, ok := r.registry[t][equipmentId]; ok {
		return val, nil
	}
	return Model{}, errors.New("not found")
}
