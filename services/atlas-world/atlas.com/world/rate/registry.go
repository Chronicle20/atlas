package rate

import (
	"sync"

	"github.com/Chronicle20/atlas-tenant"
)

type tenantData struct {
	lock       sync.RWMutex
	worldRates map[byte]Model
}

type Registry struct {
	lock    sync.RWMutex
	tenants map[tenant.Model]*tenantData
}

var rateRegistry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		rateRegistry = &Registry{
			tenants: make(map[tenant.Model]*tenantData),
		}
	})
	return rateRegistry
}

func (r *Registry) getTenantData(t tenant.Model) *tenantData {
	r.lock.RLock()
	if td, ok := r.tenants[t]; ok {
		r.lock.RUnlock()
		return td
	}
	r.lock.RUnlock()

	r.lock.Lock()
	defer r.lock.Unlock()

	if td, ok := r.tenants[t]; ok {
		return td
	}

	td := &tenantData{
		worldRates: make(map[byte]Model),
	}
	r.tenants[t] = td
	return td
}

func (r *Registry) GetWorldRates(t tenant.Model, worldId byte) Model {
	td := r.getTenantData(t)
	td.lock.RLock()
	defer td.lock.RUnlock()

	if rates, ok := td.worldRates[worldId]; ok {
		return rates
	}
	return NewModel()
}

func (r *Registry) SetWorldRate(t tenant.Model, worldId byte, rateType Type, multiplier float64) Model {
	td := r.getTenantData(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	if _, ok := td.worldRates[worldId]; !ok {
		td.worldRates[worldId] = NewModel()
	}
	td.worldRates[worldId] = td.worldRates[worldId].WithRate(rateType, multiplier)
	return td.worldRates[worldId]
}

func (r *Registry) InitWorldRates(t tenant.Model, worldId byte, rates Model) {
	td := r.getTenantData(t)
	td.lock.Lock()
	defer td.lock.Unlock()

	td.worldRates[worldId] = rates
}
