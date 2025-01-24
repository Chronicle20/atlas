package monster

import (
	"atlas-data/xml"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type MonsterString struct {
	name string
}

func (m MonsterString) Name() string {
	return m.name
}

type MonsterStringRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]MonsterString
	tenantLock map[tenant.Model]*sync.RWMutex
}

var msReg *MonsterStringRegistry
var msOnce sync.Once

func GetMonsterStringRegistry() *MonsterStringRegistry {
	msOnce.Do(func() {
		msReg = &MonsterStringRegistry{}
		msReg.registry = make(map[tenant.Model]map[uint32]MonsterString)
		msReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return msReg
}

func (r *MonsterStringRegistry) Clear(t tenant.Model) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]MonsterString)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.registry, t)
	r.registry[t] = make(map[uint32]MonsterString)
	return nil
}

func (r *MonsterStringRegistry) Init(t tenant.Model, path string) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]MonsterString)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, mxml := range exml.ChildNodes {
		id, err := strconv.Atoi(mxml.Name)
		if err != nil {
			return err
		}
		r.registry[t][uint32(id)] = MonsterString{
			name: mxml.GetString("name", "MISSINGNO"),
		}
	}
	return nil
}

func (r *MonsterStringRegistry) Read(t tenant.Model, monsterId uint32) (MonsterString, error) {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]MonsterString)
		r.lock.Unlock()
	}
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()
	if val, ok := r.registry[t][monsterId]; ok {
		return val, nil
	}
	return MonsterString{}, errors.New("not found")
}
