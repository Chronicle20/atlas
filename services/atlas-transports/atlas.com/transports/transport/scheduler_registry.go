package transport

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
)

type SchedulerRegistry struct {
	mutex    sync.RWMutex
	register map[uuid.UUID]*Scheduler
}

var schedulerRegistry *SchedulerRegistry
var schedulerRegistryOnce sync.Once

func getSchedulerRegistry() *SchedulerRegistry {
	schedulerRegistryOnce.Do(func() {
		schedulerRegistry = &SchedulerRegistry{}

		schedulerRegistry.register = make(map[uuid.UUID]*Scheduler)
	})
	return schedulerRegistry
}

func (r *SchedulerRegistry) AddTenant(t tenant.Model, scheduler *Scheduler) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.register[t.Id()] = scheduler
}

func (r *SchedulerRegistry) Get(t tenant.Model) *Scheduler {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.register[t.Id()]
}
