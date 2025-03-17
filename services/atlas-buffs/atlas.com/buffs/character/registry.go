package character

import (
	"atlas-buffs/buff"
	"atlas-buffs/buff/stat"
	"errors"
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

func (r *Registry) Apply(t tenant.Model, worldId byte, characterId uint32, sourceId int32, duration int32, changes []stat.Model) buff.Model {
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

	var m Model
	if m, ok = r.characterReg[t][characterId]; ok {
	} else {
		m = Model{
			tenant:      t,
			worldId:     worldId,
			characterId: characterId,
			buffs:       make(map[int32]buff.Model),
		}
	}
	b := buff.NewBuff(sourceId, duration, changes)
	m.buffs[sourceId] = b

	cm[characterId] = m
	cml.Unlock()
	return b
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

func (r *Registry) GetTenants() ([]tenant.Model, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	var res = make([]tenant.Model, 0)
	for t := range r.characterReg {
		res = append(res, t)
	}
	return res, nil
}

func (r *Registry) GetCharacters(t tenant.Model) []Model {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.characterReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.Lock()
	defer tl.Unlock()
	var res = make([]Model, 0)
	for _, m := range r.characterReg[t] {
		res = append(res, m)
	}
	return res
}

func (r *Registry) Cancel(t tenant.Model, characterId uint32, sourceId int32) (buff.Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.characterReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.Lock()
	defer tl.Unlock()
	var c Model
	if c, ok = r.characterReg[t][characterId]; !ok {
		return buff.Model{}, ErrNotFound
	}
	var b buff.Model
	var not = make(map[int32]buff.Model)
	for id, m := range c.buffs {
		if m.SourceId() != sourceId {
			not[id] = m
		} else {
			b = m
		}
	}
	c.buffs = not
	r.characterReg[t][characterId] = c

	if b.SourceId() != sourceId {
		return buff.Model{}, ErrNotFound
	}
	return b, nil
}

func (r *Registry) GetExpired(t tenant.Model, characterId uint32) []buff.Model {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.characterReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.Lock()
	defer tl.Unlock()
	var c Model
	if c, ok = r.characterReg[t][characterId]; !ok {
		return make([]buff.Model, 0)
	}
	var not = make(map[int32]buff.Model)
	var res = make([]buff.Model, 0)
	for id, m := range c.buffs {
		if m.Expired() {
			res = append(res, m)
		} else {
			not[id] = m
		}
	}
	c.buffs = not
	r.characterReg[t][characterId] = c
	return res
}
