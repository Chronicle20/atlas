package drop

import (
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
	"sync/atomic"
)

type dropRegistry struct {
	lock sync.RWMutex

	dropMap          map[uint32]Model
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
			dropMap:          make(map[uint32]Model),
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
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
}

func getNextUniqueId() uint32 {
	id := atomic.AddUint32(&uniqueId, 1)
	if id > 2000000000 {
		atomic.StoreUint32(&uniqueId, 1000000001)
		return 1000000001
	}
	return id
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
	currentUniqueId := getNextUniqueId()

	drop := mb.SetId(currentUniqueId).SetStatus(StatusAvailable).Build()

	d.dropMap[drop.Id()] = drop
	d.lock.Unlock()

	d.lockDrop(currentUniqueId)
	d.lockMap(mk)
	d.dropsInMap[mk] = append(d.dropsInMap[mk], drop.Id())
	d.unlockMap(mk)
	d.unlockDrop(currentUniqueId)
	return drop
}

func (d *dropRegistry) lockMap(mk mapKey) {
	d.lock.Lock()
	if _, exists := d.mapLocks[mk]; !exists {
		d.mapLocks[mk] = &sync.Mutex{}
	}
	lock := d.mapLocks[mk]
	d.lock.Unlock()
	lock.Lock()
}

func (d *dropRegistry) unlockMap(mk mapKey) {
	if lock, ok := d.mapLocks[mk]; ok {
		lock.Unlock()
	}
}

func (d *dropRegistry) lockDrop(dropId uint32) {
	d.lock.Lock()
	if _, exists := d.dropLocks[dropId]; !exists {
		d.dropLocks[dropId] = &sync.Mutex{}
	}
	lock := d.dropLocks[dropId]
	d.lock.Unlock()
	lock.Lock()
}

func (d *dropRegistry) unlockDrop(dropId uint32) {
	if lock, ok := d.dropLocks[dropId]; ok {
		lock.Unlock()
	}
}

func (d *dropRegistry) getDrop(dropId uint32) (Model, bool) {
	var drop Model
	var ok bool
	d.lock.RLock()
	drop, ok = d.dropMap[dropId]
	d.lock.RUnlock()
	return drop, ok
}

func (d *dropRegistry) CancelDropReservation(dropId uint32, characterId uint32) {
	d.lockDrop(dropId)
	defer d.unlockDrop(dropId)

	drop, ok := d.getDrop(dropId)
	if !ok {
		return
	}

	if val, ok := d.dropReservations[dropId]; ok {
		if val != characterId {
			return
		}
	} else {
		return
	}

	if drop.Status() != StatusReserved {
		return
	}

	drop = drop.CancelReservation()
	d.dropMap[drop.Id()] = drop
	delete(d.dropReservations, dropId)
}

func (d *dropRegistry) ReserveDrop(dropId uint32, characterId uint32, petSlot int8) (Model, error) {
	d.lockDrop(dropId)
	defer d.unlockDrop(dropId)

	drop, ok := d.getDrop(dropId)
	if !ok {
		return Model{}, errors.New("unable to locate drop")
	}

	if drop.Status() == StatusAvailable {
		drop = drop.Reserve(petSlot)
		d.dropMap[drop.Id()] = drop
		d.dropReservations[dropId] = characterId
		return drop, nil
	} else {
		if locker, ok := d.dropReservations[dropId]; ok && locker == characterId {
			return drop, nil
		} else {
			return Model{}, errors.New("reserved by another party")
		}
	}
}

func (d *dropRegistry) RemoveDrop(dropId uint32) (Model, error) {
	var drop Model
	d.lockDrop(dropId)

	drop, ok := d.getDrop(dropId)
	if !ok {
		d.unlockDrop(dropId)
		return Model{}, nil
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
	d.unlockDrop(dropId)
	d.unlockMap(mk)
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
	return drop, nil
}

func (d *dropRegistry) GetDropsForMap(tenant tenant.Model, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error) {
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
			drops = append(drops, drop)
		}
	}
	d.unlockMap(mk)
	return drops, nil
}

func (d *dropRegistry) GetAllDrops() []Model {
	var drops []Model
	d.lock.RLock()
	for _, drop := range d.dropMap {
		drops = append(drops, drop)
	}
	d.lock.RUnlock()
	return drops
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
