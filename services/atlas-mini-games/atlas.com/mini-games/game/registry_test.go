package game

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	return tm
}

func fieldAt(mapId uint32) field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(mapId)).Build()
}

// freshRegistry returns a Registry instance isolated from the process-wide
// singleton, so tests don't leak state into each other via GetRegistry().
func freshRegistry() *Registry {
	return &Registry{
		rooms:   make(map[tenant.Model]map[uint32]Room),
		members: make(map[tenant.Model]map[uint32]uint32),
	}
}

func TestRegistry_CreateGet_RoundTrip(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r := NewBuilder(1, 1001, fieldAt(100000000)).SetTitle("room").Build()

	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok := reg.Get(tn, r.Id())
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if got.Title() != "room" {
		t.Errorf("Get() Title() = %q, want %q", got.Title(), "room")
	}
}

func TestRegistry_Get_Missing(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)

	_, ok := reg.Get(tn, 9999)
	if ok {
		t.Errorf("Get() ok = true for missing room, want false")
	}
}

func TestRegistry_Create_RejectsDoubleRoom(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r1 := NewBuilder(1, 1001, fieldAt(100000000)).Build()

	if err := reg.Create(tn, r1); err != nil {
		t.Fatalf("Create() first room error = %v", err)
	}

	// Same owner tries to open a second room (different room id would be
	// impossible since Id()==OwnerId, but exercise the member-index check
	// directly: owner 1001 is already indexed as a member).
	r2 := NewBuilder(2, 1001, fieldAt(100000001)).Build()
	if err := reg.Create(tn, r2); err == nil {
		t.Errorf("Create() second room for same owner error = nil, want non-nil")
	}
}

func TestRegistry_Create_RejectsWhenOwnerIsVisitorElsewhere(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r1 := NewBuilder(1, 1001, fieldAt(100000000)).SetVisitorId(2002).Build()
	if err := reg.Create(tn, r1); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 2002 is a visitor in r1; trying to own a new room should be rejected.
	r2 := NewBuilder(1, 2002, fieldAt(100000001)).Build()
	if err := reg.Create(tn, r2); err == nil {
		t.Errorf("Create() for visitor-turned-owner error = nil, want non-nil")
	}
}

func TestRegistry_GetByMember_FindsOwnerAndVisitor(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r := NewBuilder(1, 1001, fieldAt(100000000)).SetVisitorId(2002).Build()
	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	owner, ok := reg.GetByMember(tn, 1001)
	if !ok || owner.Id() != r.Id() {
		t.Errorf("GetByMember(owner) = (%v, %v), want room %d", owner, ok, r.Id())
	}

	visitor, ok := reg.GetByMember(tn, 2002)
	if !ok || visitor.Id() != r.Id() {
		t.Errorf("GetByMember(visitor) = (%v, %v), want room %d", visitor, ok, r.Id())
	}

	_, ok = reg.GetByMember(tn, 9999)
	if ok {
		t.Errorf("GetByMember(stranger) ok = true, want false")
	}
}

func TestRegistry_Remove_ClearsMemberIndexForBoth(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r := NewBuilder(1, 1001, fieldAt(100000000)).SetVisitorId(2002).Build()
	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	reg.Remove(tn, r.Id())

	if _, ok := reg.Get(tn, r.Id()); ok {
		t.Errorf("Get() after Remove ok = true, want false")
	}
	if _, ok := reg.GetByMember(tn, 1001); ok {
		t.Errorf("GetByMember(owner) after Remove ok = true, want false")
	}
	if _, ok := reg.GetByMember(tn, 2002); ok {
		t.Errorf("GetByMember(visitor) after Remove ok = true, want false")
	}

	// The owner id is free again.
	r2 := NewBuilder(1, 1001, fieldAt(100000001)).Build()
	if err := reg.Create(tn, r2); err != nil {
		t.Errorf("Create() after Remove error = %v, want nil", err)
	}
}

func TestRegistry_GetInField_Filters(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	fA := fieldAt(100000000)
	fB := fieldAt(100000001)

	rA1 := NewBuilder(1, 1001, fA).Build()
	rA2 := NewBuilder(1, 1002, fA).Build()
	rB1 := NewBuilder(1, 1003, fB).Build()

	for _, r := range []Room{rA1, rA2, rB1} {
		if err := reg.Create(tn, r); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	inA := reg.GetInField(tn, fA)
	if len(inA) != 2 {
		t.Fatalf("GetInField(fA) len = %d, want 2", len(inA))
	}
	seen := map[uint32]bool{}
	for _, r := range inA {
		seen[r.Id()] = true
	}
	if !seen[1001] || !seen[1002] {
		t.Errorf("GetInField(fA) = %v, want ids 1001 and 1002", inA)
	}

	inB := reg.GetInField(tn, fB)
	if len(inB) != 1 || inB[0].Id() != 1003 {
		t.Errorf("GetInField(fB) = %v, want [1003]", inB)
	}
}

func TestRegistry_Update_SwapVisibleToNextGet(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r := NewBuilder(1, 1001, fieldAt(100000000)).SetVisitorId(2002).Build()
	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := reg.Update(tn, r.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetInProgress(true).SetVisitorId(3003).Build(), nil
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !updated.InProgress() {
		t.Errorf("Update() returned room InProgress() = false, want true")
	}

	got, ok := reg.Get(tn, r.Id())
	if !ok {
		t.Fatalf("Get() after Update ok = false")
	}
	if !got.InProgress() {
		t.Errorf("Get() after Update InProgress() = false, want true")
	}
	if got.VisitorId() != 3003 {
		t.Errorf("Get() after Update VisitorId() = %d, want 3003", got.VisitorId())
	}

	// Member index must reflect the new visitor, not the old one.
	if _, ok := reg.GetByMember(tn, 2002); ok {
		t.Errorf("GetByMember(old visitor) after Update ok = true, want false")
	}
	newVisitorRoom, ok := reg.GetByMember(tn, 3003)
	if !ok || newVisitorRoom.Id() != r.Id() {
		t.Errorf("GetByMember(new visitor) = (%v, %v), want room %d", newVisitorRoom, ok, r.Id())
	}
}

func TestRegistry_Update_VisitorLeaves_ClearsVisitorIndexKeepsOwner(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r := NewBuilder(1, 1001, fieldAt(100000000)).SetVisitorId(2002).Build()
	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := reg.Update(tn, r.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetVisitorId(0).Build(), nil
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if _, ok := reg.GetByMember(tn, 2002); ok {
		t.Errorf("GetByMember(departed visitor) ok = true, want false")
	}
	ownerRoom, ok := reg.GetByMember(tn, 1001)
	if !ok || ownerRoom.Id() != r.Id() {
		t.Errorf("GetByMember(owner) = (%v, %v), want room %d", ownerRoom, ok, r.Id())
	}

	// The departed visitor is free to open their own room.
	r2 := NewBuilder(1, 2002, fieldAt(100000001)).Build()
	if err := reg.Create(tn, r2); err != nil {
		t.Errorf("Create() for departed visitor error = %v, want nil", err)
	}
}

func TestRegistry_Update_ErrorLeavesRoomUntouched(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	r := NewBuilder(1, 1001, fieldAt(100000000)).SetTitle("before").Build()
	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sentinel := errors.New("boom")
	_, err := reg.Update(tn, r.Id(), func(cur Room) (Room, error) {
		return Clone(cur).SetTitle("after").Build(), sentinel
	})
	if err == nil {
		t.Fatalf("Update() error = nil, want non-nil")
	}

	got, ok := reg.Get(tn, r.Id())
	if !ok {
		t.Fatalf("Get() ok = false")
	}
	if got.Title() != "before" {
		t.Errorf("Get() Title() = %q after failed Update, want %q", got.Title(), "before")
	}
}

func TestRegistry_Update_MissingRoom(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)

	_, err := reg.Update(tn, 9999, func(cur Room) (Room, error) {
		return cur, nil
	})
	if err == nil {
		t.Errorf("Update() on missing room error = nil, want non-nil")
	}
}

func TestRegistry_Race_UpdateGetGetInField(t *testing.T) {
	reg := freshRegistry()
	tn := testTenant(t)
	f := fieldAt(100000000)
	r := NewBuilder(1, 1001, f).SetVisitorId(2002).Build()
	if err := reg.Create(tn, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = reg.Update(tn, r.Id(), func(cur Room) (Room, error) {
					return Clone(cur).SetOwnerScore(int32(n*100 + j)).Build(), nil
				})
				_, _ = reg.Get(tn, r.Id())
				_ = reg.GetInField(tn, f)
				_, _ = reg.GetByMember(tn, 2002)
			}
		}(i)
	}
	wg.Wait()
}
