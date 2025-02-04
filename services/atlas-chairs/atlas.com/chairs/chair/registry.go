package chair

import (
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

func (r *Registry) Set(t tenant.Model, id uint32, chair Model) (Model, error) {
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
	r.characterReg[t][id] = chair
	return chair, nil
}

func (r *Registry) Clear(t tenant.Model, id uint32) error {
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
	delete(r.characterReg[t], id)
	return nil
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
