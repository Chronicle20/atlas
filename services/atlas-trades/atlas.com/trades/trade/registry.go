package trade

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	// ErrOwnerHasRoom is returned by Create when a participant already occupies
	// another trade room for the tenant.
	ErrOwnerHasRoom = errors.New("trade: character already has a room")
	// ErrRoomNotFound is returned by Update when the room id is unknown.
	ErrRoomNotFound = errors.New("trade: room not found")
	// ErrRoomFull is returned when a second character tries to enter an
	// already-paired room.
	ErrRoomFull = errors.New("trade: room is full")
	// ErrRoomFrozen is returned when a staging or transition action arrives
	// after the room left the state that permits it (FR-3.6, design §3.2).
	ErrRoomFrozen = errors.New("trade: room is frozen")
	// ErrHandleInUse is returned when a room would claim a wire handle another
	// room of the same tenant already owns. Handles default to the owner's
	// character id, which the occupancy check already makes unique, so this
	// only fires for a caller-supplied SetHandle. Without the guard the newer
	// room would silently overwrite handles[t][h] and its removal would delete
	// the older room's entry, stranding that room unreachable by GetByHandle.
	ErrHandleInUse = errors.New("trade: handle already in use")
)

// Registry is the tenant-partitioned in-memory store of trade rooms. One
// RWMutex guards all three maps; the member and handle indexes are maintained
// ONLY inside Create/Update/Remove, always alongside the room mutation, under
// the write lock. The registry is process-local, which is why atlas-trades runs
// replicas: 1 (design §9).
//
// Two uniqueness invariants hold for every tenant, enforced by Create and
// Update: a character occupies at most one room, and a wire handle identifies
// at most one room.
type Registry struct {
	mutex   sync.RWMutex
	rooms   map[tenant.Model]map[uuid.UUID]Room
	members map[tenant.Model]map[character.Id]uuid.UUID
	handles map[tenant.Model]map[uint32]uuid.UUID
}

var (
	registry *Registry
	once     sync.Once
)

func newRoomMap() map[tenant.Model]map[uuid.UUID]Room {
	return make(map[tenant.Model]map[uuid.UUID]Room)
}

func newMemberMap() map[tenant.Model]map[character.Id]uuid.UUID {
	return make(map[tenant.Model]map[character.Id]uuid.UUID)
}

func newHandleMap() map[tenant.Model]map[uint32]uuid.UUID {
	return make(map[tenant.Model]map[uint32]uuid.UUID)
}

// GetRegistry returns the process-wide Registry singleton.
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	})
	return registry
}

// Create registers r for tenant t. It fails with ErrOwnerHasRoom if any of r's
// participants already occupies a room, and with ErrHandleInUse if another room
// already owns r's wire handle.
func (reg *Registry) Create(t tenant.Model, r Room) error {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	if err := reg.checkIndexable(t, r, uuid.Nil); err != nil {
		return err
	}

	if reg.rooms[t] == nil {
		reg.rooms[t] = make(map[uuid.UUID]Room)
	}
	if reg.members[t] == nil {
		reg.members[t] = make(map[character.Id]uuid.UUID)
	}
	if reg.handles[t] == nil {
		reg.handles[t] = make(map[uint32]uuid.UUID)
	}

	reg.rooms[t][r.Id()] = r
	reg.index(t, r)
	return nil
}

// Get returns the room identified by id for tenant t.
func (reg *Registry) Get(t tenant.Model, id uuid.UUID) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	r, ok := reg.rooms[t][id]
	return r, ok
}

// GetByMember resolves the room characterId occupies, as either side.
func (reg *Registry) GetByMember(t tenant.Model, characterId character.Id) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	id, ok := reg.members[t][characterId]
	if !ok {
		return Room{}, false
	}
	r, ok := reg.rooms[t][id]
	return r, ok
}

// GetByHandle resolves a room from the uint32 wire serial an invite carries.
func (reg *Registry) GetByHandle(t tenant.Model, handle uint32) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	id, ok := reg.handles[t][handle]
	if !ok {
		return Room{}, false
	}
	r, ok := reg.rooms[t][id]
	return r, ok
}

// All returns every live room for tenant t (the REST list read).
func (reg *Registry) All(t tenant.Model) []Room {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	out := make([]Room, 0, len(reg.rooms[t]))
	for _, r := range reg.rooms[t] {
		out = append(out, r)
	}
	return out
}

// Update mutates the room under a single write lock: fn receives the current
// Room and returns its replacement. A non-nil error from fn leaves the room
// untouched and is returned as-is — this is how state transitions are made
// compare-and-set (design §12), so two simultaneous confirms cannot both
// trigger settlement.
//
// fn RUNS WHILE THE REGISTRY WRITE LOCK IS HELD, and must therefore be pure:
// no REST calls, no Kafka emits, no re-entry into the Registry (any Get/Create/
// Update/Remove from inside fn self-deadlocks), no blocking of any kind. Decide
// the transition inside fn; perform the I/O it implies after Update returns.
//
// A replacement that would collide with another room's member or handle index
// is rejected (ErrOwnerHasRoom / ErrHandleInUse) before anything is mutated.
func (reg *Registry) Update(t tenant.Model, id uuid.UUID, fn func(Room) (Room, error)) (Room, error) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	cur, ok := reg.rooms[t][id]
	if !ok {
		return Room{}, ErrRoomNotFound
	}

	updated, err := fn(cur)
	if err != nil {
		return Room{}, err
	}

	if err = reg.checkIndexable(t, updated, cur.Id()); err != nil {
		return Room{}, err
	}

	reg.deindex(t, cur)
	delete(reg.rooms[t], cur.Id())
	reg.rooms[t][updated.Id()] = updated
	reg.index(t, updated)
	return updated, nil
}

// Remove deletes the room and clears every index entry it owns. A missing id is
// a no-op.
func (reg *Registry) Remove(t tenant.Model, id uuid.UUID) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	r, ok := reg.rooms[t][id]
	if !ok {
		return
	}
	delete(reg.rooms[t], id)
	reg.deindex(t, r)
}

// checkIndexable reports whether r can own its member and handle index entries
// for tenant t. Entries already held by selfId belong to the room being
// replaced and are not collisions; pass uuid.Nil when there is no such room.
// Callers hold the write lock. It reads only, so a rejection needs no rollback.
func (reg *Registry) checkIndexable(t tenant.Model, r Room, selfId uuid.UUID) error {
	for _, p := range r.participants {
		if held, ok := reg.members[t][p.characterId]; ok && held != selfId {
			return ErrOwnerHasRoom
		}
	}
	if held, ok := reg.handles[t][r.Handle()]; ok && held != selfId {
		return ErrHandleInUse
	}
	return nil
}

// index records every participant and the wire handle. Callers hold the write
// lock and have ensured the per-tenant maps are non-nil.
func (reg *Registry) index(t tenant.Model, r Room) {
	for _, p := range r.participants {
		reg.members[t][p.characterId] = r.Id()
	}
	reg.handles[t][r.Handle()] = r.Id()
}

// deindex removes every participant and the wire handle. Callers hold the write
// lock.
func (reg *Registry) deindex(t tenant.Model, r Room) {
	for _, p := range r.participants {
		delete(reg.members[t], p.characterId)
	}
	delete(reg.handles[t], r.Handle())
}
