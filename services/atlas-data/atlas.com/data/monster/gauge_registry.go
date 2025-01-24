package monster

import (
	"atlas-data/xml"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type MonsterGaugeRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]bool
	tenantLock map[tenant.Model]*sync.RWMutex
}

var mgReg *MonsterGaugeRegistry
var mgOnce sync.Once

func GetMonsterGaugeRegistry() *MonsterGaugeRegistry {
	mgOnce.Do(func() {
		mgReg = &MonsterGaugeRegistry{}
		mgReg.registry = make(map[tenant.Model]map[uint32]bool)
		mgReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return mgReg
}

func (r *MonsterGaugeRegistry) Clear(t tenant.Model) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]bool)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.registry, t)
	r.registry[t] = make(map[uint32]bool)
	return nil
}

func (r *MonsterGaugeRegistry) Init(t tenant.Model, path string) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]bool)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	exml, err := xml.Read(path)
	if err != nil {
		return err
	}
	d, err := exml.ChildByName("MobGage/Mob")
	if err != nil {
		return err
	}

	for _, mxml := range d.CanvasNodes {
		idStr := mxml.Name
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return err
		}
		r.registry[t][uint32(id)] = true
	}
	return nil
}

func (r *MonsterGaugeRegistry) Read(t tenant.Model, monsterId uint32) (bool, error) {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]bool)
		r.lock.Unlock()
	}
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()
	if val, ok := r.registry[t][monsterId]; ok {
		return val, nil
	}
	return false, errors.New("not found")
}
