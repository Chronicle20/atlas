package monster

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type Registry struct {
	registry map[string]map[uint32]Model
	mutex    sync.RWMutex
}

var once sync.Once
var registry *Registry

func GetRegistry() *Registry {
	once.Do(func() {
		registry = initRegistry()
	})
	return registry
}

func initRegistry() *Registry {
	s := &Registry{make(map[string]map[uint32]Model), sync.RWMutex{}}
	return s
}

func (r *Registry) GetMonster(tenant tenant.Model, monsterId uint32) (Model, error) {
	var m Model
	m, err := r.attemptRead(tenant.Id().String(), monsterId)
	if err == nil {
		return m, nil
	}
	m, err = r.attemptLoad(tenant, tenant.Id().String(), monsterId)
	if err == nil {
		return m, nil
	}

	rid := tenant.Region() + "-" + strconv.Itoa(int(tenant.MajorVersion())) + "." + strconv.Itoa(int(tenant.MinorVersion()))
	m, err = r.attemptRead(rid, monsterId)
	if err == nil {
		return m, nil
	}
	return r.attemptLoad(tenant, rid, monsterId)
}

func (r *Registry) attemptRead(id string, monsterId uint32) (Model, error) {
	var m Model
	r.mutex.RLock()
	if t, ok := r.registry[id]; ok {
		if m, ok = t[monsterId]; ok {
			r.mutex.RUnlock()
			return m, nil
		}
	}
	r.mutex.RUnlock()
	return m, errors.New("not found")
}

func (r *Registry) attemptLoad(tenant tenant.Model, id string, monsterId uint32) (Model, error) {
	var m Model
	r.mutex.Lock()
	eq, err := Read(tenant, monsterId)
	if err != nil {
		r.mutex.Unlock()
		return Model{}, err
	}

	m = *eq

	if _, ok := r.registry[id]; !ok {
		r.registry[id] = make(map[uint32]Model)
	}
	r.registry[id][monsterId] = m
	r.mutex.Unlock()
	return m, nil
}
