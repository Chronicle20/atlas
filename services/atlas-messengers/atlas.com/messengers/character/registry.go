package character

import (
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

var ErrNotFound = errors.New("not found")

type Registry struct {
	lock         sync.Mutex
	characterReg map[tenant.Model]map[uint32]Model
	tenantLock   map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) Create(t tenant.Model, worldId world.Id, channelId channel.Id, id uint32, name string) Model {
	r.lock.Lock()

	var cm map[uint32]Model
	var cml *sync.RWMutex
	var ok bool
	if cm, ok = r.characterReg[t]; ok {
		cml = r.tenantLock[t]
	} else {
		cm = make(map[uint32]Model)
		cml = &sync.RWMutex{}
	}
	r.characterReg[t] = cm
	r.tenantLock[t] = cml
	r.lock.Unlock()

	cml.Lock()

	m := Model{
		tenantId:    t.Id(),
		id:          id,
		name:        name,
		worldId:     worldId,
		channelId:   channelId,
		messengerId: 0,
		online:      false,
	}
	cm[id] = m
	cml.Unlock()
	return m
}

func (r *Registry) Get(t tenant.Model, id uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.characterReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	if m, ok := r.characterReg[t][id]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

func (r *Registry) Update(t tenant.Model, id uint32, updaters ...func(m Model) Model) Model {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	m := r.characterReg[t][id]
	for _, updater := range updaters {
		m = updater(m)
	}
	r.characterReg[t][id] = m
	return m
}
