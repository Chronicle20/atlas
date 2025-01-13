package party

import (
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Registry struct {
	lock          sync.Mutex
	tenantPartyId map[tenant.Model]uint32

	partyReg   map[tenant.Model]map[uint32]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenantPartyId = make(map[tenant.Model]uint32)
		registry.partyReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) Create(t tenant.Model, leaderId uint32) Model {
	var partyId uint32
	var partyReg map[uint32]Model
	var tenantLock *sync.RWMutex
	var ok bool

	r.lock.Lock()
	if partyId, ok = r.tenantPartyId[t]; ok {
		partyId += 1
		partyReg = r.partyReg[t]
		tenantLock = r.tenantLock[t]
	} else {
		partyId = StartPartyId
		partyReg = make(map[uint32]Model)
		tenantLock = &sync.RWMutex{}
	}
	r.tenantPartyId[t] = partyId
	r.partyReg[t] = partyReg
	r.tenantLock[t] = tenantLock
	r.lock.Unlock()

	m := Model{
		tenantId: t.Id(),
		id:       partyId,
		leaderId: leaderId,
		members:  make([]uint32, 0),
	}
	m.members = append(m.members, leaderId)

	tenantLock.Lock()
	partyReg[partyId] = m
	tenantLock.Unlock()
	return m
}

func (r *Registry) GetAll(t tenant.Model) []Model {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.partyReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	var res = make([]Model, 0)
	for _, v := range r.partyReg[t] {
		res = append(res, v)
	}
	return res
}

func (r *Registry) Get(t tenant.Model, partyId uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.partyReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	if m, ok := r.partyReg[t][partyId]; ok {
		return m, nil
	}
	return Model{}, ErrNotFound
}

func (r *Registry) Update(t tenant.Model, id uint32, updaters ...func(m Model) Model) (Model, error) {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	m := r.partyReg[t][id]
	for _, updater := range updaters {
		m = updater(m)
	}

	if len(m.members) > 6 {
		return Model{}, ErrAtCapacity
	}

	r.partyReg[t][id] = m
	return m, nil
}

func (r *Registry) Remove(t tenant.Model, partyId uint32) {
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.partyReg[t], partyId)
}
