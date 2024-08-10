package account

import (
	"atlas-login/tenant"
	"sync"
)

type Key struct {
	Tenant tenant.Model
	Id     uint32
}

type Registry struct {
	mutex    sync.RWMutex
	accounts map[Key]bool
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.accounts = make(map[Key]bool)
	})
	return registry
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
