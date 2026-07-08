package game

import (
	"errors"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrOwnerHasRoom is returned by Create when the room's owner is already a
// member (owner or visitor) of another room for the tenant.
var ErrOwnerHasRoom = errors.New("game: owner already has a room")

// ErrRoomNotFound is returned by Update when roomId does not exist for the
// tenant.
var ErrRoomNotFound = errors.New("game: room not found")

// Registry is the tenant-partitioned in-memory store of mini-game rooms.
// rooms holds the current Room snapshot per (tenant, roomId); members
// indexes characterId -> roomId for BOTH the owner and the visitor, so
// GetByMember and Create's double-room check are O(1). Both maps are
// guarded by one RWMutex; the member index is maintained only inside
// Create/Update/Remove, always alongside the room mutation, under the
// write lock — it is never read or written anywhere else.
type Registry struct {
	mutex   sync.RWMutex
	rooms   map[tenant.Model]map[uint32]Room
	members map[tenant.Model]map[uint32]uint32
}

var registry *Registry
var once sync.Once

// GetRegistry returns the process-wide Registry singleton.
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{
			rooms:   make(map[tenant.Model]map[uint32]Room),
			members: make(map[tenant.Model]map[uint32]uint32),
		}
	})
	return registry
}

// Create adds r for tenant t. It fails with ErrOwnerHasRoom if r's owner is
// already a member of any room for t (as owner or visitor) — a character
// can occupy at most one room at a time.
func (reg *Registry) Create(t tenant.Model, r Room) error {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	if _, ok := reg.members[t][r.OwnerId()]; ok {
		return ErrOwnerHasRoom
	}

	if reg.rooms[t] == nil {
		reg.rooms[t] = make(map[uint32]Room)
	}
	if reg.members[t] == nil {
		reg.members[t] = make(map[uint32]uint32)
	}

	reg.rooms[t][r.Id()] = r
	reg.indexMembers(t, r)
	return nil
}

// Get returns the room identified by roomId for tenant t.
func (reg *Registry) Get(t tenant.Model, roomId uint32) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	r, ok := reg.rooms[t][roomId]
	return r, ok
}

// GetByMember returns the room characterId currently occupies, as either
// owner or visitor, for tenant t.
func (reg *Registry) GetByMember(t tenant.Model, characterId uint32) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	roomId, ok := reg.members[t][characterId]
	if !ok {
		return Room{}, false
	}
	r, ok := reg.rooms[t][roomId]
	return r, ok
}

// GetInField returns every room for tenant t located in field f.
func (reg *Registry) GetInField(t tenant.Model, f field.Model) []Room {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	var out []Room
	for _, r := range reg.rooms[t] {
		if r.Field().Equals(f) {
			out = append(out, r)
		}
	}
	return out
}

// Update mutates the room identified by roomId for tenant t under a single
// write lock: fn receives the current Room and returns its replacement. If
// fn returns a non-nil error, the room is left untouched and the error is
// returned as-is. On success, the room is swapped and the member index is
// rebuilt from the updated room's owner/visitor ids, since either may have
// changed (e.g. a visitor joining or leaving).
func (reg *Registry) Update(t tenant.Model, roomId uint32, fn func(Room) (Room, error)) (Room, error) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	cur, ok := reg.rooms[t][roomId]
	if !ok {
		return Room{}, ErrRoomNotFound
	}

	updated, err := fn(cur)
	if err != nil {
		return Room{}, err
	}

	delete(reg.rooms[t], roomId)
	reg.rooms[t][updated.Id()] = updated

	reg.deindexMembers(t, cur)
	reg.indexMembers(t, updated)
	return updated, nil
}

// Remove deletes the room identified by roomId for tenant t and clears its
// owner/visitor from the member index. A missing roomId is a no-op.
func (reg *Registry) Remove(t tenant.Model, roomId uint32) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	r, ok := reg.rooms[t][roomId]
	if !ok {
		return
	}
	delete(reg.rooms[t], roomId)
	reg.deindexMembers(t, r)
}

// indexMembers records r's owner and (if present) visitor in the member
// index for tenant t. Callers must hold the write lock and must have
// already ensured reg.members[t] is non-nil.
func (reg *Registry) indexMembers(t tenant.Model, r Room) {
	reg.members[t][r.OwnerId()] = r.Id()
	if r.VisitorId() != 0 {
		reg.members[t][r.VisitorId()] = r.Id()
	}
}

// deindexMembers removes r's owner and (if present) visitor from the
// member index for tenant t. Callers must hold the write lock.
func (reg *Registry) deindexMembers(t tenant.Model, r Room) {
	delete(reg.members[t], r.OwnerId())
	if r.VisitorId() != 0 {
		delete(reg.members[t], r.VisitorId())
	}
}
