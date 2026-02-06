package account

import (
	"sync"

	"github.com/Chronicle20/atlas-tenant"
)

type Key struct {
	Tenant tenant.Model
	Id     uint32
}

func KeyForTenantFunc(t tenant.Model) func(m Model) Key {
	return func(m Model) Key {
		return Key{Tenant: t, Id: m.Id()}
	}
}

type Registry struct {
	mutex    sync.RWMutex
	accounts map[Key]bool
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.accounts = make(map[Key]bool)
	})
	return registry
}

func (r *Registry) Init(as map[Key]bool) {
	r.mutex.Lock()
	for k, b := range as {
		r.accounts[k] = b
	}
	r.mutex.Unlock()
}

func (r *Registry) Login(Key Key) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.accounts[Key] = true
}

func (r *Registry) Logout(key Key) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.accounts[key] = false
}

func (r *Registry) LoggedIn(key Key) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if val, ok := r.accounts[key]; ok {
		return val
	}
	return false
}
