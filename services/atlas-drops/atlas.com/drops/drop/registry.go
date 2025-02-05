package drop

import (
	"errors"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
)

type dropRegistry struct {
	lock sync.RWMutex

	dropMap          map[uint32]*Model
	dropReservations map[uint32]uint32

	dropLocks map[uint32]*sync.Mutex

	mapLocks   map[mapKey]*sync.Mutex
	dropsInMap map[mapKey][]uint32
}

var registry *dropRegistry
var once sync.Once

var uniqueId = uint32(1000000001)

func GetRegistry() *dropRegistry {
	once.Do(func() {
		registry = &dropRegistry{
			lock:             sync.RWMutex{},
			dropMap:          make(map[uint32]*Model),
			dropLocks:        make(map[uint32]*sync.Mutex),
			mapLocks:         make(map[mapKey]*sync.Mutex),
			dropsInMap:       make(map[mapKey][]uint32),
			dropReservations: make(map[uint32]uint32),
		}
	})
	return registry
}

type mapKey struct {
	tenantId  uuid.UUID
	worldId   byte
	channelId byte
	mapId     uint32
}

func (d *dropRegistry) CreateDrop(mb *ModelBuilder) Model {
	t := mb.Tenant()
	mk := mapKey{
		tenantId:  t.Id(),
		worldId:   mb.WorldId(),
		channelId: mb.ChannelId(),
		mapId:     mb.MapId(),
	}

	d.lock.Lock()
	ids := existingIds(d.dropMap)
	currentUniqueId := uniqueId
	for contains(ids, currentUniqueId) {
		currentUniqueId = currentUniqueId + 1
		if currentUniqueId > 2000000000 {
			currentUniqueId = 1000000001
		}
		uniqueId = currentUniqueId
	}

	drop := mb.SetId(currentUniqueId).SetStatus(StatusAvailable).Build()

	d.dropMap[drop.Id()] = &drop
	d.lock.Unlock()

	d.lockDrop(currentUniqueId)
	d.lockMap(mk)
	d.dropsInMap[mk] = append(d.dropsInMap[mk], drop.Id())
	d.unlockMap(mk)
	d.unlockDrop(currentUniqueId)
	return drop
}

func (d *dropRegistry) lockMap(mk mapKey) {
	if lock, ok := d.mapLocks[mk]; ok {
		lock.Lock()
	} else {
		d.lock.Lock()
		mapMutex := sync.Mutex{}
		d.mapLocks[mk] = &mapMutex
		mapMutex.Lock()
		d.lock.Unlock()
	}
}

func (d *dropRegistry) unlockMap(mk mapKey) {
	if lock, ok := d.mapLocks[mk]; ok {
		lock.Unlock()
	}
}

func (d *dropRegistry) lockDrop(dropId uint32) {
	if lock, ok := d.dropLocks[dropId]; ok {
		lock.Lock()
	} else {
		d.lock.Lock()
		dropMutex := sync.Mutex{}
		d.dropLocks[dropId] = &dropMutex
		dropMutex.Lock()
		d.lock.Unlock()
	}
}

func (d *dropRegistry) unlockDrop(dropId uint32) {
	if lock, ok := d.dropLocks[dropId]; ok {
		lock.Unlock()
	}
}

func (d *dropRegistry) getDrop(dropId uint32) (*Model, bool) {
	var drop *Model
	var ok bool
	d.lock.RLock()
	drop, ok = d.dropMap[dropId]
	d.lock.RUnlock()
	return drop, ok
}

func (d *dropRegistry) CancelDropReservation(dropId uint32, characterId uint32) {
	d.lockDrop(dropId)

	drop, ok := d.getDrop(dropId)
	if !ok {
		d.unlockDrop(dropId)
		return
	}

	if val, ok := d.dropReservations[dropId]; ok {
		if val != characterId {
			d.unlockDrop(dropId)
			return
		}
	} else {
		d.unlockDrop(dropId)
		return
	}

	if drop.Status() != StatusReserved {
		d.unlockDrop(dropId)
		return
	}

	drop.CancelReservation()
	delete(d.dropReservations, dropId)
	d.unlockDrop(dropId)
}

func (d *dropRegistry) ReserveDrop(dropId uint32, characterId uint32) error {
	d.lockDrop(dropId)

	drop, ok := d.getDrop(dropId)

	if !ok {
		d.unlockDrop(dropId)
		return errors.New("unable to locate drop")
	}

	if drop.Status() == StatusAvailable {
		drop.Reserve()
		d.dropReservations[dropId] = characterId
		d.unlockDrop(dropId)
		return nil
	} else {
		if locker, ok := d.dropReservations[dropId]; ok && locker == characterId {
			d.unlockDrop(dropId)
			return nil
		} else {
			d.unlockDrop(dropId)
			return errors.New("reserved by another party")
		}
	}
}

func (d *dropRegistry) RemoveDrop(dropId uint32) (*Model, error) {
	var drop *Model
	d.lockDrop(dropId)

	drop, ok := d.getDrop(dropId)
	if !ok {
		d.unlockDrop(dropId)
		return nil, nil
	}

	d.lock.Lock()
	delete(d.dropMap, dropId)
	delete(d.dropReservations, dropId)
	d.lock.Unlock()

	t := drop.Tenant()
	mk := mapKey{
		tenantId:  t.Id(),
		worldId:   drop.WorldId(),
		channelId: drop.ChannelId(),
		mapId:     drop.MapId(),
	}

	d.lockMap(mk)
	if _, ok := d.dropsInMap[mk]; ok {
		index := indexOf(dropId, d.dropsInMap[mk])
		if index >= 0 && index < len(d.dropsInMap[mk]) {
			d.dropsInMap[mk] = remove(d.dropsInMap[mk], index)
		}
	}
	d.unlockMap(mk)

	d.unlockDrop(dropId)
	return drop, nil
}

func (d *dropRegistry) GetDrop(dropId uint32) (Model, error) {
	d.lockDrop(dropId)
	drop, ok := d.getDrop(dropId)
	if !ok {
		d.unlockDrop(dropId)
		return Model{}, errors.New("drop not found")
	}
	d.unlockDrop(dropId)
	return *drop, nil
}

func (d *dropRegistry) GetDropsForMap(tenant tenant.Model, worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	mk := mapKey{
		tenantId:  tenant.Id(),
		worldId:   worldId,
		channelId: channelId,
		mapId:     mapId,
	}
	drops := make([]Model, 0)
	d.lockMap(mk)
	for _, dropId := range d.dropsInMap[mk] {
		if drop, ok := d.getDrop(dropId); ok {
			drops = append(drops, *drop)
		}
	}
	d.unlockMap(mk)
	return drops, nil
}

func (d *dropRegistry) GetAllDrops() []Model {
	var drops []Model
	d.lock.RLock()
	for _, drop := range d.dropMap {
		drops = append(drops, *drop)
	}
	d.lock.RUnlock()
	return drops
}

func existingIds(drops map[uint32]*Model) []uint32 {
	var ids []uint32
	for i := range drops {
		ids = append(ids, i)
	}
	return ids
}

func contains(ids []uint32, id uint32) bool {
	for _, element := range ids {
		if element == id {
			return true
		}
	}
	return false
}

func indexOf(uniqueId uint32, data []uint32) int {
	for k, v := range data {
		if uniqueId == v {
			return k
		}
	}
	return -1 //not found.
}

func remove(s []uint32, i int) []uint32 {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
