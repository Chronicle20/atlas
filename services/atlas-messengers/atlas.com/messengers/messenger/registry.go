package messenger

import (
	"sync"

	"github.com/Chronicle20/atlas-tenant"
)

type Registry struct {
	lock              sync.Mutex
	tenantMessengerId map[tenant.Model]uint32

	messengerReg map[tenant.Model]map[uint32]Model
	tenantLock   map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenantMessengerId = make(map[tenant.Model]uint32)
		registry.messengerReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) Create(t tenant.Model, characterId uint32) Model {
	var messengerId uint32
	var messengerReg map[uint32]Model
	var tenantLock *sync.RWMutex
	var ok bool

	r.lock.Lock()
	if messengerId, ok = r.tenantMessengerId[t]; ok {
		messengerId += 1
		messengerReg = r.messengerReg[t]
		tenantLock = r.tenantLock[t]
	} else {
		messengerId = StartMessengerId
		messengerReg = make(map[uint32]Model)
		tenantLock = &sync.RWMutex{}
	}
	r.tenantMessengerId[t] = messengerId
	r.messengerReg[t] = messengerReg
	r.tenantLock[t] = tenantLock
	r.lock.Unlock()

	m, _ := NewBuilder().
		SetTenantId(t.Id()).
		SetId(messengerId).
		AddMember(characterId, 0).
		Build()

	tenantLock.Lock()
	messengerReg[messengerId] = m
	tenantLock.Unlock()
	return m
}

func (r *Registry) GetAll(t tenant.Model) []Model {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.messengerReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	var res = make([]Model, 0)
	for _, v := range r.messengerReg[t] {
		res = append(res, v)
	}
	return res
}

func (r *Registry) Get(t tenant.Model, messengerId uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.messengerReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	if m, ok := r.messengerReg[t][messengerId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

func (r *Registry) Update(t tenant.Model, id uint32, updaters ...func(m Model) Model) (Model, error) {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	m := r.messengerReg[t][id]
	for _, updater := range updaters {
		m = updater(m)
	}

	if len(m.members) > MaxMembers {
		return Model{}, ErrAtCapacity
	}

	r.messengerReg[t][id] = m
	return m, nil
}

func (r *Registry) Remove(t tenant.Model, messengerId uint32) {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.messengerReg[t], messengerId)
}
