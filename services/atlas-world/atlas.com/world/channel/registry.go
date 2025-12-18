package channel

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Registry struct {
	lock       sync.Mutex
	registry   map[tenant.Model]map[byte]map[byte]Model
	tenantLock map[tenant.Model]*sync.RWMutex
}

var channelRegistry *Registry
var once sync.Once

var ErrChannelNotFound = errors.New("channel not found")

func GetChannelRegistry() *Registry {
	once.Do(func() {
		channelRegistry = &Registry{
			registry:   make(map[tenant.Model]map[byte]map[byte]Model),
			tenantLock: make(map[tenant.Model]*sync.RWMutex),
		}
	})
	return channelRegistry
}

func (r *Registry) ensureTenantLock(t tenant.Model) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.tenantLock[t]; !ok {
		r.tenantLock[t] = &sync.RWMutex{}
		r.registry[t] = make(map[byte]map[byte]Model)
	}
}

func (r *Registry) Register(t tenant.Model, m Model) Model {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	if _, ok := r.registry[t][m.WorldId()]; !ok {
		r.registry[t][m.WorldId()] = make(map[byte]Model)
	}
	r.registry[t][m.WorldId()][m.channelId] = m
	return m
}

func (r *Registry) ChannelServers(t tenant.Model) []Model {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	results := make([]Model, 0)
	for _, w := range r.registry[t] {
		for _, c := range w {
			results = append(results, c)
		}
	}
	return results
}

func (r *Registry) ChannelServer(t tenant.Model, worldId byte, channelId byte) (Model, error) {
	r.ensureTenantLock(t)
	r.tenantLock[t].RLock()
	defer r.tenantLock[t].RUnlock()

	var ok bool
	var result Model
	if _, ok = r.registry[t][worldId]; !ok {
		return result, ErrChannelNotFound
	}
	if result, ok = r.registry[t][worldId][channelId]; !ok {
		return result, ErrChannelNotFound
	}
	return result, nil
}

func (r *Registry) RemoveByWorldAndChannel(t tenant.Model, worldId byte, channelId byte) error {
	r.ensureTenantLock(t)
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	if _, ok := r.registry[t][worldId]; !ok {
		return ErrChannelNotFound
	}
	if _, ok := r.registry[t][worldId][channelId]; !ok {
		return ErrChannelNotFound
	}
	delete(r.registry[t][worldId], channelId)
	return nil
}

func (r *Registry) Tenants() []tenant.Model {
	r.lock.Lock()
	defer r.lock.Unlock()
	results := make([]tenant.Model, 0)
	for t := range r.registry {
		results = append(results, t)
	}
	return results
}
