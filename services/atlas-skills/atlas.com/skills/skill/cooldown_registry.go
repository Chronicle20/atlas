package skill

import (
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
)

var ErrNotFound = errors.New("not found")

type Registry struct {
	lock         sync.Mutex
	characterReg map[tenant.Model]map[uint32]map[uint32]time.Time
	tenantLock   map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterReg = make(map[tenant.Model]map[uint32]map[uint32]time.Time)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) Apply(t tenant.Model, characterId uint32, skillId uint32, cooldown uint32) error {
	r.lock.Lock()

	var cm map[uint32]map[uint32]time.Time
	var cml *sync.RWMutex
	var ok bool
	if cm, ok = r.characterReg[t]; ok {
		cml = r.tenantLock[t]
	} else {
		cm = make(map[uint32]map[uint32]time.Time)
		cml = &sync.RWMutex{}
	}
	r.characterReg[t] = cm
	r.tenantLock[t] = cml
	r.lock.Unlock()

	cml.Lock()

	if _, ok = r.characterReg[t][characterId]; !ok {
		r.characterReg[t][characterId] = make(map[uint32]time.Time)
	}
	r.characterReg[t][characterId][skillId] = time.Now().Add(time.Duration(cooldown) * time.Second)

	cml.Unlock()
	return nil
}

func (r *Registry) Get(t tenant.Model, characterId uint32, skillId uint32) (time.Time, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.characterReg[t] = make(map[uint32]map[uint32]time.Time)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	if _, ok = r.characterReg[t][characterId]; !ok {
		return time.Time{}, ErrNotFound
	}
	var cooldownExpiresAt time.Time
	if cooldownExpiresAt, ok = r.characterReg[t][characterId][skillId]; !ok {
		return time.Time{}, ErrNotFound
	}

	return cooldownExpiresAt, nil
}

func (r *Registry) ClearAll(t tenant.Model, characterId uint32) error {
	r.lock.Lock()

	var cm map[uint32]map[uint32]time.Time
	var cml *sync.RWMutex
	var ok bool
	if cm, ok = r.characterReg[t]; ok {
		cml = r.tenantLock[t]
	} else {
		cm = make(map[uint32]map[uint32]time.Time)
		cml = &sync.RWMutex{}
	}
	r.characterReg[t] = cm
	r.tenantLock[t] = cml
	r.lock.Unlock()

	cml.Lock()

	delete(r.characterReg[t], characterId)

	cml.Unlock()
	return nil
}

func (r *Registry) Clear(t tenant.Model, characterId uint32, skillId uint32) error {
	r.lock.Lock()

	var cm map[uint32]map[uint32]time.Time
	var cml *sync.RWMutex
	var ok bool
	if cm, ok = r.characterReg[t]; ok {
		cml = r.tenantLock[t]
	} else {
		cm = make(map[uint32]map[uint32]time.Time)
		cml = &sync.RWMutex{}
	}
	r.characterReg[t] = cm
	r.tenantLock[t] = cml
	r.lock.Unlock()

	cml.Lock()

	if _, ok = r.characterReg[t][characterId]; !ok {
		r.characterReg[t][characterId] = make(map[uint32]time.Time)
	}
	delete(r.characterReg[t][characterId], skillId)

	cml.Unlock()
	return nil
}

type CooldownHolder struct {
	tenant            tenant.Model
	characterId       uint32
	skillId           uint32
	cooldownExpiresAt time.Time
}

func (h CooldownHolder) CooldownExpiresAt() time.Time {
	return h.cooldownExpiresAt
}

func (h CooldownHolder) Tenant() tenant.Model {
	return h.tenant
}

func (h CooldownHolder) CharacterId() uint32 {
	return h.characterId
}

func (h CooldownHolder) SkillId() uint32 {
	return h.skillId
}

func (r *Registry) GetAll() []CooldownHolder {
	r.lock.Lock()
	defer r.lock.Unlock()

	res := make([]CooldownHolder, 0)
	for t := range r.characterReg {
		r.tenantLock[t].RLock()
		for c := range r.characterReg[t] {
			for sid, cet := range r.characterReg[t][c] {
				res = append(res, CooldownHolder{
					tenant:            t,
					characterId:       c,
					skillId:           sid,
					cooldownExpiresAt: cet,
				})
			}
		}
		r.tenantLock[t].RUnlock()
	}
	return res
}
