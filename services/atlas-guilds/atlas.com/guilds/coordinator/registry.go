package coordinator

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
	"time"
)

type Registry struct {
	lock         sync.Mutex
	characterReg map[tenant.Model]map[uint32]uuid.UUID
	agreementReg map[uuid.UUID]Model
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterReg = make(map[tenant.Model]map[uint32]uuid.UUID)
		registry.agreementReg = make(map[uuid.UUID]Model)
	})
	return registry
}

func (r *Registry) Initiate(t tenant.Model, worldId byte, channelId byte, name string, leaderId uint32, members []uint32) error {
	r.lock.Lock()
	if _, ok := r.characterReg[t]; !ok {
		r.characterReg[t] = make(map[uint32]uuid.UUID)
	}

	agreementId := uuid.New()
	for _, m := range members {
		if val, ok := r.characterReg[t][m]; ok && val != uuid.Nil {
			r.lock.Unlock()
			return errors.New("already attempting guild creation")
		}
		r.characterReg[t][m] = agreementId
	}
	rm := make(map[uint32]bool)
	rm[leaderId] = true

	r.agreementReg[agreementId] = Model{
		tenant:    t,
		worldId:   worldId,
		channelId: channelId,
		leaderId:  leaderId,
		name:      name,
		requests:  members,
		responses: rm,
		age:       time.Now(),
	}
	r.lock.Unlock()
	return nil
}

func (r *Registry) Respond(t tenant.Model, characterId uint32, agree bool) (Model, error) {
	r.lock.Lock()
	agreementId := r.characterReg[t][characterId]
	g := r.agreementReg[agreementId]
	if agree {
		g = g.Agree(characterId)
		r.lock.Unlock()
		return g, nil
	} else {
		delete(r.agreementReg, agreementId)
		for _, m := range g.requests {
			r.characterReg[t][m] = uuid.Nil
		}
		r.lock.Unlock()
		return Model{}, nil
	}
}

func (r *Registry) GetExpired(timeout time.Duration) ([]Model, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	now := time.Now()
	var results = make([]Model, 0)
	for _, g := range r.agreementReg {
		if now.Sub(g.Age()) > timeout {
			results = append(results, g)
		}
	}
	return results, nil
}
