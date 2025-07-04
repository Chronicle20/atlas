package expression

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
	"time"
)

type Registry struct {
	lock          sync.Mutex
	expressionReg map[tenant.Model]map[uint32]Model
	tenantLock    map[tenant.Model]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.expressionReg = make(map[tenant.Model]map[uint32]Model)
		registry.tenantLock = make(map[tenant.Model]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) add(t tenant.Model, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) Model {
	r.lock.Lock()
	if _, ok := r.expressionReg[t]; !ok {
		r.expressionReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = &sync.RWMutex{}
	}
	r.lock.Unlock()

	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()

	expiration := time.Now().Add(time.Second * time.Duration(5))

	e := Model{
		tenant:      t,
		characterId: characterId,
		worldId:     worldId,
		channelId:   channelId,
		mapId:       mapId,
		expression:  expression,
		expiration:  expiration,
	}

	r.expressionReg[t][characterId] = e
	return e
}

func (r *Registry) popExpired() []Model {
	var results = make([]Model, 0)
	now := time.Now()
	r.lock.Lock()
	defer r.lock.Unlock()
	for t, cm := range r.expressionReg {
		r.tenantLock[t].Lock()
		for id, m := range cm {
			if now.Sub(m.Expiration()) > 0 {
				results = append(results, m)
				delete(r.expressionReg[t], id)
			}
		}
		r.tenantLock[t].Unlock()
	}
	return results
}

func (r *Registry) clear(t tenant.Model, characterId uint32) {
	if _, ok := r.tenantLock[t]; !ok {
		r.lock.Lock()
		r.expressionReg[t] = make(map[uint32]Model)
		r.tenantLock[t] = &sync.RWMutex{}
		r.lock.Unlock()
	}
	r.tenantLock[t].Lock()
	defer r.tenantLock[t].Unlock()
	delete(r.expressionReg[t], characterId)
}
