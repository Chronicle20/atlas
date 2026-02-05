package channel

import (
	"errors"
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type tenantData struct {
	lock     sync.RWMutex
	channels map[world.Id]map[channel.Id]Model
}

type Registry struct {
	lock    sync.RWMutex
	tenants map[tenant.Model]*tenantData
}

var channelRegistry *Registry
var once sync.Once

var ErrChannelNotFound = errors.New("channel not found")

func GetChannelRegistry() *Registry {
	once.Do(func() {
		channelRegistry = &Registry{
			tenants: make(map[tenant.Model]*tenantData),
		}
	})
	return channelRegistry
}

func (r *Registry) getTenantData(t tenant.Model) *tenantData {
	// Try read lock first for existing tenant
	r.lock.RLock()
	if td, ok := r.tenants[t]; ok {
		r.lock.RUnlock()
		return td
	}
	r.lock.RUnlock()

	// Need write lock to create new tenant
	r.lock.Lock()
	defer r.lock.Unlock()

	// Double-check after acquiring write lock
	if td, ok := r.tenants[t]; ok {
		return td
	}

	td := &tenantData{
		channels: make(map[world.Id]map[channel.Id]Model),
	}
	r.tenants[t] = td
	return td
}

func (r *Registry) Register(t tenant.Model, m Model) Model {
	td := r.getTenantData(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	if _, ok := td.channels[m.WorldId()]; !ok {
		td.channels[m.WorldId()] = make(map[channel.Id]Model)
	}
	td.channels[m.WorldId()][m.channelId] = m
	return m
}

func (r *Registry) ChannelServers(t tenant.Model) []Model {
	td := r.getTenantData(t)
	td.lock.RLock()
	defer td.lock.RUnlock()

	results := make([]Model, 0)
	for _, w := range td.channels {
		for _, c := range w {
			results = append(results, c)
		}
	}
	return results
}

func (r *Registry) ChannelServer(t tenant.Model, worldId world.Id, channelId channel.Id) (Model, error) {
	td := r.getTenantData(t)
	td.lock.RLock()
	defer td.lock.RUnlock()

	var ok bool
	var result Model
	if _, ok = td.channels[worldId]; !ok {
		return result, ErrChannelNotFound
	}
	if result, ok = td.channels[worldId][channelId]; !ok {
		return result, ErrChannelNotFound
	}
	return result, nil
}

func (r *Registry) RemoveByWorldAndChannel(t tenant.Model, worldId world.Id, channelId channel.Id) error {
	td := r.getTenantData(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	if _, ok := td.channels[worldId]; !ok {
		return ErrChannelNotFound
	}
	if _, ok := td.channels[worldId][channelId]; !ok {
		return ErrChannelNotFound
	}
	delete(td.channels[worldId], channelId)
	return nil
}

func (r *Registry) Tenants() []tenant.Model {
	r.lock.RLock()
	defer r.lock.RUnlock()
	results := make([]tenant.Model, 0)
	for t := range r.tenants {
		results = append(results, t)
	}
	return results
}
