package invite

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
	"time"
)

type Registry struct {
	lock           sync.Mutex
	tenantInviteId map[tenant.Model]uint32
	inviteReg      map[tenant.Model]map[uint32]map[string][]Model
	tenantLock     map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.tenantInviteId = make(map[tenant.Model]uint32)
		registry.inviteReg = make(map[tenant.Model]map[uint32]map[string][]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) Create(t tenant.Model, originatorId uint32, worldId byte, targetId uint32, inviteType string, referenceId uint32) Model {
	var inviteId uint32
	var inviteReg map[uint32]map[string][]Model
	var tenantLock *sync.RWMutex
	var ok bool

	r.lock.Lock()
	if inviteId, ok = r.tenantInviteId[t]; ok {
		inviteId += 1
		inviteReg = r.inviteReg[t]
		tenantLock = r.tenantLock[t]
	} else {
		inviteId = StartInviteId
		inviteReg = make(map[uint32]map[string][]Model)
		tenantLock = &sync.RWMutex{}
	}
	r.tenantInviteId[t] = inviteId
	r.inviteReg[t] = inviteReg
	r.tenantLock[t] = tenantLock
	r.lock.Unlock()

	m := Model{
		tenant:       t,
		id:           inviteId,
		inviteType:   inviteType,
		referenceId:  referenceId,
		originatorId: originatorId,
		targetId:     targetId,
		worldId:      worldId,
		age:          time.Now(),
	}

	tenantLock.Lock()
	defer tenantLock.Unlock()
	if _, ok = r.inviteReg[t][targetId]; !ok {
		r.inviteReg[t][targetId] = make(map[string][]Model)
	}

	if _, ok = r.inviteReg[t][targetId][inviteType]; !ok {
		r.inviteReg[t][targetId][inviteType] = make([]Model, 0)
	}

	for _, i := range r.inviteReg[t][targetId][inviteType] {
		if i.ReferenceId() == referenceId {
			return i
		}
	}
	r.inviteReg[t][targetId][inviteType] = append(r.inviteReg[t][targetId][inviteType], m)
	return m
}

func (r *Registry) GetByOriginator(t tenant.Model, actorId uint32, inviteType string, originatorId uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.inviteReg[t] = make(map[uint32]map[string][]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	var tenReg map[uint32]map[string][]Model
	if tenReg, ok = r.inviteReg[t]; ok {
		var charReg map[string][]Model
		if charReg, ok = tenReg[actorId]; ok {
			var invReg []Model
			if invReg, ok = charReg[inviteType]; ok {
				for _, i := range invReg {
					if i.OriginatorId() == originatorId {
						return i, nil
					}
				}
			}
		}
	}
	return Model{}, errors.New("not found")
}

func (r *Registry) GetByReference(t tenant.Model, actorId uint32, inviteType string, referenceId uint32) (Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.inviteReg[t] = make(map[uint32]map[string][]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	var tenReg map[uint32]map[string][]Model
	if tenReg, ok = r.inviteReg[t]; ok {
		var charReg map[string][]Model
		if charReg, ok = tenReg[actorId]; ok {
			var invReg []Model
			if invReg, ok = charReg[inviteType]; ok {
				for _, i := range invReg {
					if i.ReferenceId() == referenceId {
						return i, nil
					}
				}
			}
		}
	}
	return Model{}, errors.New("not found")
}

func (r *Registry) GetForCharacter(t tenant.Model, characterId uint32) ([]Model, error) {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.inviteReg[t] = make(map[uint32]map[string][]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.RLock()
	defer tl.RUnlock()
	var results = make([]Model, 0)

	var tenReg map[uint32]map[string][]Model
	if tenReg, ok = r.inviteReg[t]; ok {
		var charReg map[string][]Model
		if charReg, ok = tenReg[characterId]; ok {
			for _, v := range charReg {
				results = append(results, v...)
			}
		}
	}
	return results, nil

}

func (r *Registry) Delete(t tenant.Model, actorId uint32, inviteType string, originatorId uint32) error {
	var tl *sync.RWMutex
	var ok bool
	if tl, ok = r.tenantLock[t]; !ok {
		r.lock.Lock()
		tl = &sync.RWMutex{}
		r.inviteReg[t] = make(map[uint32]map[string][]Model)
		r.tenantLock[t] = tl
		r.lock.Unlock()
	}

	tl.Lock()
	defer tl.Unlock()
	var tenReg map[uint32]map[string][]Model
	if tenReg, ok = r.inviteReg[t]; ok {
		var charReg map[string][]Model
		if charReg, ok = tenReg[actorId]; ok {
			var invReg []Model
			if invReg, ok = charReg[inviteType]; ok {
				var found = false
				var remain = make([]Model, 0)
				for _, i := range invReg {
					if i.OriginatorId() != originatorId {
						remain = append(remain, i)
					} else {
						found = true
					}
				}
				r.inviteReg[t][actorId][inviteType] = remain
				if found {
					return nil
				}
			}
		}
	}
	return errors.New("not found")
}

func (r *Registry) GetExpired(timeout time.Duration) ([]Model, error) {
	var results = make([]Model, 0)
	for k, v := range r.inviteReg {
		if tl, ok := r.tenantLock[k]; ok {
			tl.RLock()
			for _, cir := range v {
				for _, is := range cir {
					for _, i := range is {
						if i.Expired(timeout) {
							results = append(results, i)
						}
					}
				}
			}
			tl.RUnlock()
		}
	}
	return results, nil
}
