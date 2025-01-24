package monster

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type MonsterModelRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var mmReg *MonsterModelRegistry
var mmOnce sync.Once

func GetMonsterModelRegistry() *MonsterModelRegistry {
	mmOnce.Do(func() {
		mmReg = &MonsterModelRegistry{}
		mmReg.registry = make(map[tenant.Model]map[uint32]Model)
		mmReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return mmReg
}

func (r *MonsterModelRegistry) Add(t tenant.Model, m Model) error {
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

func (r *MonsterModelRegistry) Get(t tenant.Model, monsterId uint32) (Model, error) {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]Model)
		r.lock.Unlock()
	}

	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	if val, ok := r.registry[t][monsterId]; ok {
		return val, nil
	}
	return Model{}, errors.New("not found")
}
