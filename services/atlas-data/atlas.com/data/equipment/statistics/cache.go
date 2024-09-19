package statistics

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"strconv"
	"sync"
)

type equipmentCache struct {
	mutex     sync.RWMutex
	equipment map[string]map[uint32]Model
}

var cache *equipmentCache
var once sync.Once

func GetEquipmentCache() *equipmentCache {
	once.Do(func() {
		cache = &equipmentCache{
			mutex:     sync.RWMutex{},
			equipment: make(map[string]map[uint32]Model),
		}
	})
	return cache
}

func (e *equipmentCache) attemptRead(id string, itemId uint32) (Model, error) {
	var equipment Model
	e.mutex.RLock()
	if t, ok := e.equipment[id]; ok {
		if equipment, ok = t[itemId]; ok {
			e.mutex.RUnlock()
			return equipment, nil
		}
	}
	e.mutex.RUnlock()
	return equipment, errors.New("not found")
}

func (e *equipmentCache) attemptLoad(tenant tenant.Model, id string, itemId uint32) (Model, error) {
	var equipment Model
	e.mutex.Lock()
	eq, err := Read(tenant, itemId)
	if err != nil {
		e.mutex.Unlock()
		return Model{}, err
	}

	equipment = *eq

	if _, ok := e.equipment[id]; !ok {
		e.equipment[id] = make(map[uint32]Model)
	}
	e.equipment[id][itemId] = equipment
	e.mutex.Unlock()
	return equipment, nil
}

func (e *equipmentCache) GetEquipment(tenant tenant.Model, itemId uint32) (Model, error) {
	var equipment Model
	equipment, err := e.attemptRead(tenant.Id().String(), itemId)
	if err == nil {
		return equipment, nil
	}
	equipment, err = e.attemptLoad(tenant, tenant.Id().String(), itemId)
	if err == nil {
		return equipment, nil
	}

	rid := tenant.Region() + "-" + strconv.Itoa(int(tenant.MajorVersion())) + "." + strconv.Itoa(int(tenant.MinorVersion()))
	equipment, err = e.attemptRead(rid, itemId)
	if err == nil {
		return equipment, nil
	}
	return e.attemptLoad(tenant, rid, itemId)
}
