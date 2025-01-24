package npc

import (
	"atlas-data/xml"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type NpcString struct {
	name string
}

func (m NpcString) Name() string {
	return m.name
}

type NpcStringRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]NpcString
	tenantLock map[tenant.Model]*sync.RWMutex
}

var nsReg *NpcStringRegistry
var nsOnce sync.Once

func GetNpcStringRegistry() *NpcStringRegistry {
	nsOnce.Do(func() {
		nsReg = &NpcStringRegistry{}
		nsReg.registry = make(map[tenant.Model]map[uint32]NpcString)
		nsReg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return nsReg
}

func (r *NpcStringRegistry) Clear(t tenant.Model) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]NpcString)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.registry, t)
	r.registry[t] = make(map[uint32]NpcString)
	return nil
}

func (r *NpcStringRegistry) Init(t tenant.Model, path string) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]NpcString)
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
		r.registry[t][uint32(id)] = NpcString{
			name: mxml.GetString("name", "MISSINGNO"),
		}
	}
	return nil
}

func (r *NpcStringRegistry) Read(t tenant.Model, npcId uint32) (NpcString, error) {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]NpcString)
		r.lock.Unlock()
	}
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()
	if val, ok := r.registry[t][npcId]; ok {
		return val, nil
	}
	return NpcString{}, errors.New("not found")
}
