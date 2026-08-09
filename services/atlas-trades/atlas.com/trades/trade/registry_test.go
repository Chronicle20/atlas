package trade

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestRegistryRejectsSecondRoomForOwner pins the authoritative single-room
// invariant (design §2.1): atlas-channel's cross-family check is best effort,
// but a character may hold at most one TRADE room, enforced here.
func TestRegistryRejectsSecondRoomForOwner(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	first := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	if err := reg.Create(tm, first); err != nil {
		t.Fatalf("first create: %v", err)
	}

	second := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	if err := reg.Create(tm, second); err != ErrOwnerHasRoom {
		t.Fatalf("second create: got %v, want ErrOwnerHasRoom", err)
	}
}

// TestRegistryIndexesBothParticipants pins that GetByMember resolves for the
// invited character too, not just the owner — the EXIT/logout teardown path
// looks a room up by whichever side acted.
func TestRegistryIndexesBothParticipants(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").Build()
	if err := reg.Create(tm, room); err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, id := range []uint32{100, 200} {
		if _, ok := reg.GetByMember(tm, id); !ok {
			t.Errorf("GetByMember(%d): not found", id)
		}
	}
}

// TestRegistryRemoveClearsEveryIndex pins that teardown drops the room, both
// member entries and the wire handle — a stale index leaves the character
// permanently unable to open another trade.
func TestRegistryRemoveClearsEveryIndex(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").Build()
	_ = reg.Create(tm, room)
	reg.Remove(tm, room.Id())

	if _, ok := reg.Get(tm, room.Id()); ok {
		t.Error("room still present after Remove")
	}
	for _, id := range []uint32{100, 200} {
		if _, ok := reg.GetByMember(tm, id); ok {
			t.Errorf("member index still holds character %d", id)
		}
	}
	if _, ok := reg.GetByHandle(tm, 100); ok {
		t.Error("handle index still holds 100")
	}
}

// TestRegistryUpdateIsCompareAndSetOnState pins design §12: two simultaneous
// confirms must not both drive the room into SETTLING.
func TestRegistryUpdateIsCompareAndSetOnState(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)
	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").SetState(StateOpen).Build()
	_ = reg.Create(tm, room)

	transition := func(r Room) (Room, error) {
		if r.State() != StateOpen {
			return Room{}, ErrRoomFrozen
		}
		return r.WithState(StateSettling), nil
	}

	if _, err := reg.Update(tm, room.Id(), transition); err != nil {
		t.Fatalf("first transition: %v", err)
	}
	if _, err := reg.Update(tm, room.Id(), transition); err != ErrRoomFrozen {
		t.Fatalf("second transition: got %v, want ErrRoomFrozen", err)
	}
}

// TestRegistryTenantIsolation pins that tenant A cannot see tenant B's room.
func TestRegistryTenantIsolation(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	a, b := testTenant(t), testOtherTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	_ = reg.Create(a, room)

	if _, ok := reg.Get(b, room.Id()); ok {
		t.Error("tenant B can see tenant A's room")
	}
	if _, ok := reg.GetByMember(b, 100); ok {
		t.Error("tenant B can resolve tenant A's member")
	}
	if err := reg.Create(b, NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()); err != nil {
		t.Errorf("tenant B blocked by tenant A's occupancy: %v", err)
	}
}

// TestRegistryUpdateOnMissingRoom pins that Update reports ErrRoomNotFound
// rather than inserting a room the registry never created.
func TestRegistryUpdateOnMissingRoom(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	_, err := reg.Update(tm, uuid.New(), func(r Room) (Room, error) { return r, nil })
	if err != ErrRoomNotFound {
		t.Fatalf("update missing room: got %v, want ErrRoomNotFound", err)
	}
}

// TestRegistryAllIsTenantScoped pins that the REST list read returns only the
// asking tenant's rooms.
func TestRegistryAllIsTenantScoped(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	a, b := testTenant(t), testOtherTenant(t)

	roomA1 := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	roomA2 := NewBuilder(3, 300, "Other", testField(t)).SetHandle(300).Build()
	roomB := NewBuilder(3, 500, "Foreign", testField(t)).SetHandle(500).Build()
	_ = reg.Create(a, roomA1)
	_ = reg.Create(a, roomA2)
	_ = reg.Create(b, roomB)

	all := reg.All(a)
	if len(all) != 2 {
		t.Fatalf("All(a): got %d rooms, want 2", len(all))
	}
	for _, r := range all {
		if r.Id() == roomB.Id() {
			t.Error("All(a) leaked tenant B's room")
		}
	}
}

// TestRegistryUpdateReindexesNewParticipants pins that a room which gains a
// visitor through Update becomes resolvable by that visitor's character id —
// the invite-accepted path seats position 1 via Update, not Create.
func TestRegistryUpdateReindexesNewParticipants(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	_ = reg.Create(tm, room)

	if _, ok := reg.GetByMember(tm, 200); ok {
		t.Fatal("visitor indexed before it was seated")
	}

	_, err := reg.Update(tm, room.Id(), func(r Room) (Room, error) {
		return NewBuilder(r.RoomType(), r.OwnerId(), "Owner", r.Field()).
			SetId(r.Id()).SetHandle(r.Handle()).SetState(StateOpen).SetVisitor(200, "Guest").Build(), nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if _, ok := reg.GetByMember(tm, 200); !ok {
		t.Error("GetByMember(200): not found after the visitor was seated")
	}
}

// TestRegistryConcurrentAccess exercises every lock path from many goroutines
// at once so `-race` can prove the RWMutex covers each of them, and asserts
// that exactly one of N racing transitions wins the compare-and-set.
func TestRegistryConcurrentAccess(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").SetState(StateOpen).Build()
	if err := reg.Create(tm, room); err != nil {
		t.Fatalf("create: %v", err)
	}

	const racers = 32
	var wg sync.WaitGroup
	var mu sync.Mutex
	won := 0

	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Get(tm, room.Id())
			reg.GetByMember(tm, 200)
			reg.GetByHandle(tm, 100)
			reg.All(tm)
			_, err := reg.Update(tm, room.Id(), func(r Room) (Room, error) {
				if r.State() != StateOpen {
					return Room{}, ErrRoomFrozen
				}
				return r.WithState(StateSettling), nil
			})
			if err == nil {
				mu.Lock()
				won++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if won != 1 {
		t.Fatalf("compare-and-set winners: got %d, want 1", won)
	}
	got, ok := reg.Get(tm, room.Id())
	if !ok {
		t.Fatal("room vanished during the race")
	}
	if got.State() != StateSettling {
		t.Fatalf("final state: got %v, want %v", got.State(), StateSettling)
	}
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	return tm
}

func testOtherTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 87, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	return tm
}

func testField(t *testing.T) field.Model {
	t.Helper()
	return field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
}
