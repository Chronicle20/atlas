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

// getOrCreateTenantMaps returns the character map and lock for a tenant,
// creating them if they don't exist. This method is thread-safe.
func (r *Registry) getOrCreateTenantMaps(t tenant.Model) (map[uint32]Model, *sync.RWMutex) {
	r.lock.Lock()
	defer r.lock.Unlock()

	cm, ok := r.characterReg[t]
	if !ok {
		cm = make(map[uint32]Model)
		r.characterReg[t] = cm
	}

	cml, ok := r.tenantLock[t]
	if !ok {
		cml = &sync.RWMutex{}
		r.tenantLock[t] = cml
	}

	return cm, cml
}

func (r *Registry) Apply(t tenant.Model, worldId byte, characterId uint32, sourceId int32, duration int32, changes []stat.Model) (buff.Model, error) {
	b, err := buff.NewBuff(sourceId, duration, changes)
	if err != nil {
		return buff.Model{}, err
	}

	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	var m Model
	var ok bool
	if m, ok = cm[characterId]; !ok {
		m = Model{
			tenant:      t,
			worldId:     worldId,
			characterId: characterId,
			buffs:       make(map[int32]buff.Model),
		}
	}
	m.buffs[sourceId] = b

	cm[characterId] = m
	return b, nil
}

func (r *Registry) Get(t tenant.Model, id uint32) (Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	if m, ok := cm[id]; ok {
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
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.RLock()
	defer cml.RUnlock()

	res := make([]Model, 0, len(cm))
	for _, m := range cm {
		res = append(res, m)
	}
	return res
}

func (r *Registry) Cancel(t tenant.Model, characterId uint32, sourceId int32) (buff.Model, error) {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	c, ok := cm[characterId]
	if !ok {
		return buff.Model{}, ErrNotFound
	}

	var b buff.Model
	var found bool
	not := make(map[int32]buff.Model)
	for id, m := range c.buffs {
		if m.SourceId() != sourceId {
			not[id] = m
		} else {
			b = m
			found = true
		}
	}
	c.buffs = not
	cm[characterId] = c

	if !found {
		return buff.Model{}, ErrNotFound
	}
	return b, nil
}

func (r *Registry) GetExpired(t tenant.Model, characterId uint32) []buff.Model {
	cm, cml := r.getOrCreateTenantMaps(t)

	cml.Lock()
	defer cml.Unlock()

	c, ok := cm[characterId]
	if !ok {
		return make([]buff.Model, 0)
	}

	not := make(map[int32]buff.Model)
	res := make([]buff.Model, 0)
	for id, m := range c.buffs {
		if m.Expired() {
			res = append(res, m)
		} else {
			not[id] = m
		}
	}
	c.buffs = not
	cm[characterId] = c
	return res
}

// ResetForTesting clears all registry state. Only for use in tests.
func (r *Registry) ResetForTesting() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.characterReg = make(map[tenant.Model]map[uint32]Model)
	r.tenantLock = make(map[tenant.Model]*sync.RWMutex)
}
