package _map

import (
	"atlas-data/xml"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type MapString struct {
	mapName    string
	streetName string
}

func (m MapString) MapName() string {
	return m.mapName
}

func (m MapString) StreetName() string {
	return m.streetName
}

type MapStringRegistry struct {
	lock sync.Mutex

	registry   map[tenant.Model]map[uint32]MapString
	tenantLock map[tenant.Model]*sync.RWMutex
}

var msRg *MapStringRegistry
var msOnce sync.Once

func GetMapStringRegistry() *MapStringRegistry {
	msOnce.Do(func() {
		msRg = &MapStringRegistry{}
		msRg.registry = make(map[tenant.Model]map[uint32]MapString)
		msRg.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return msRg
}

func (r *MapStringRegistry) Clear(t tenant.Model) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]MapString)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.registry, t)
	r.registry[t] = make(map[uint32]MapString)
	return nil
}

func (r *MapStringRegistry) Init(t tenant.Model, path string) error {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]MapString)
		r.lock.Unlock()
	}

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	if len(r.registry[t]) != 0 {
		return nil
	}

	exml, err := xml.Read(path)
	if err != nil {
		return err
	}

	for _, cat := range exml.ChildNodes {
		for _, mxml := range cat.ChildNodes {
			id, err := strconv.Atoi(mxml.Name)
			if err != nil {
				return err
			}
			r.registry[t][uint32(id)] = MapString{
				mapName:    mxml.GetString("mapName", ""),
				streetName: mxml.GetString("streetName", ""),
			}
		}
	}
	return nil
}

func (r *MapStringRegistry) Read(t tenant.Model, mapId uint32) (MapString, error) {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[uint32]MapString)
		r.lock.Unlock()
	}
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()
	if val, ok := r.registry[t][mapId]; ok {
		return val, nil
	}
	return MapString{}, errors.New("not found")
}
