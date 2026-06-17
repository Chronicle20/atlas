package door

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// -----------------------------------------------------------------------------
// Test seams
// -----------------------------------------------------------------------------

// fakeResolver returns canned town/slot inputs so Spawn unit-tests without REST.
type fakeResolver struct {
	partyId     uint32
	townMapId   _map.Id
	doorPortals []TownPortal
	members     []character.Id
	durationMs  int32
	resolveErr  error
}

func (f fakeResolver) PartyIdFor(_ context.Context, _ character.Id) (uint32, error) {
	return f.partyId, nil
}

func (f fakeResolver) ResolveSpawn(_ context.Context, _ field.Model, ownerCharacterId character.Id, partyId uint32, _ skill.Id, _ byte) (spawnPlan, error) {
	if f.resolveErr != nil {
		return spawnPlan{}, f.resolveErr
	}
	slot := ComputeSlot(partyId, f.members, ownerCharacterId)
	wireId, tx, ty, _ := ResolveTownPortal(f.doorPortals, slot, defaultTownX, defaultTownY)
	return spawnPlan{townMapId: f.townMapId, slot: slot, townPortalId: wireId, townX: tx, townY: ty, durationMs: f.durationMs}, nil
}

// counterAllocator is a deterministic id allocator stub. It hands out
// base, base+1, base+2, ... and can be configured to error on the Nth Allocate
// call (1-based). Released ids are recorded for assertions.
type counterAllocator struct {
	next     uint32
	calls    int
	errOn    int // 1-based call index that should error; 0 = never
	released []uint32
}

func (a *counterAllocator) Allocate(_ context.Context, _ tenant.Model) (uint32, error) {
	a.calls++
	if a.errOn != 0 && a.calls == a.errOn {
		return 0, errors.New("alloc failed")
	}
	id := a.next
	a.next++
	return id, nil
}

func (a *counterAllocator) Release(_ context.Context, _ tenant.Model, id uint32) {
	a.released = append(a.released, id)
}

// fakeEmit captures (topic, decoded event Type) for each emitted message, plus
// the raw value so tests can decode further fields (partyId/forCharacterId/...).
type fakeEmit struct {
	topics []string
	types  []string
	values [][]byte
}

func (e *fakeEmit) emit(topic string, p model.Provider[[]kafka.Message]) error {
	msgs, err := p()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		var env struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(msg.Value, &env)
		e.topics = append(e.topics, topic)
		e.types = append(e.types, env.Type)
		e.values = append(e.values, msg.Value)
	}
	return nil
}

func newTestProcessor(t *testing.T, res resolver, alloc allocator, em *fakeEmit) (*ProcessorImpl, tenant.Model, context.Context) {
	t.Helper()
	ten, ctx := newTestTenant()
	GetRegistry().Clear(ctx)
	return &ProcessorImpl{
		l:     logrus.New(),
		ctx:   ctx,
		t:     ten,
		emit:  em.emit,
		res:   res,
		alloc: alloc,
	}, ten, ctx
}

// -----------------------------------------------------------------------------
// Spawn
// -----------------------------------------------------------------------------

func TestSpawnCreatesPairAndEmitsCreated(t *testing.T) {
	res := fakeResolver{partyId: 0, townMapId: _map.Id(104000000), durationMs: 120000}
	alloc := &counterAllocator{next: 1_000_001}
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, res, alloc, em)

	f := field.NewBuilder(1, 2, 100000000).Build()
	m, err := p.Spawn(f, 42, 9101000, 1, 100, 200)
	if err != nil {
		t.Fatalf("Spawn error: %v", err)
	}

	if m.AreaDoorId() != 1_000_001 || m.TownDoorId() != 1_000_002 {
		t.Fatalf("expected oid pair 1000001/1000002, got %d/%d", m.AreaDoorId(), m.TownDoorId())
	}
	if m.PairId() != m.AreaDoorId() {
		t.Fatalf("pairId should equal areaDoorId, got %d vs %d", m.PairId(), m.AreaDoorId())
	}
	if m.AreaX() != 100 || m.AreaY() != 200 {
		t.Fatalf("area pos mismatch: %d,%d", m.AreaX(), m.AreaY())
	}
	if m.ExpiresAt().IsZero() {
		t.Fatalf("expected non-zero expiry for positive duration")
	}

	// persisted
	got, err := GetRegistry().Get(ctx, ten, 1_000_001)
	if err != nil {
		t.Fatalf("door not persisted: %v", err)
	}
	if got.OwnerCharacterId() != 42 {
		t.Fatalf("persisted owner mismatch: %d", got.OwnerCharacterId())
	}

	// exactly one CREATED emitted (recast cleanup found nothing)
	if len(em.types) != 1 || em.types[0] != EventDoorStatusCreated {
		t.Fatalf("expected [CREATED], got %v", em.types)
	}
	if em.topics[0] != EnvEventTopicDoorStatus {
		t.Fatalf("expected topic %s, got %s", EnvEventTopicDoorStatus, em.topics[0])
	}
}

func TestSpawnRecastReplacesExisting(t *testing.T) {
	res := fakeResolver{partyId: 0, townMapId: _map.Id(104000000), durationMs: 120000}
	alloc := &counterAllocator{next: 1_000_001}
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, res, alloc, em)

	f := field.NewBuilder(1, 2, 100000000).Build()

	// Pre-seed an existing owner door (simulating a prior cast).
	prior := NewBuilder().
		SetAreaDoorId(500_001).SetTownDoorId(500_002).
		SetOwnerCharacterId(42).SetPartyId(0).SetField(f).
		SetTownMapId(_map.Id(104000000)).SetSlot(0).Build()
	if err := GetRegistry().Put(ctx, ten, prior); err != nil {
		t.Fatalf("pre-seed Put: %v", err)
	}

	m, err := p.Spawn(f, 42, 9101000, 1, 100, 200)
	if err != nil {
		t.Fatalf("Spawn error: %v", err)
	}

	// The new door uses freshly allocated oids.
	if m.AreaDoorId() != 1_000_001 {
		t.Fatalf("expected new areaDoorId 1000001, got %d", m.AreaDoorId())
	}

	// Prior door is gone.
	if _, err := GetRegistry().Get(ctx, ten, 500_001); err == nil {
		t.Fatalf("expected prior door 500001 to be removed")
	}

	// Owner now has exactly the new door.
	doors, err := GetRegistry().GetByOwner(ctx, ten, 42)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(doors) != 1 || doors[0].AreaDoorId() != 1_000_001 {
		t.Fatalf("expected only new door for owner, got %+v", doors)
	}

	// Emit order: REMOVED (recast) BEFORE CREATED.
	if len(em.types) != 2 {
		t.Fatalf("expected 2 events, got %v", em.types)
	}
	if em.types[0] != EventDoorStatusRemoved || em.types[1] != EventDoorStatusCreated {
		t.Fatalf("expected [REMOVED, CREATED], got %v", em.types)
	}
}

func TestSpawnFailsCleanlyOnAllocError(t *testing.T) {
	res := fakeResolver{partyId: 0, townMapId: _map.Id(104000000), durationMs: 120000}
	// Error on the SECOND allocate (town oid).
	alloc := &counterAllocator{next: 1_000_001, errOn: 2}
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, res, alloc, em)

	f := field.NewBuilder(1, 2, 100000000).Build()
	_, err := p.Spawn(f, 42, 9101000, 1, 100, 200)
	if err == nil {
		t.Fatalf("expected Spawn to fail on town oid alloc error")
	}

	// No CREATED emitted.
	for _, ty := range em.types {
		if ty == EventDoorStatusCreated {
			t.Fatalf("CREATED must not be emitted on alloc failure; got %v", em.types)
		}
	}

	// No persist (area oid 1000001 must not be in the registry).
	if _, gerr := GetRegistry().Get(ctx, ten, 1_000_001); gerr == nil {
		t.Fatalf("door must not be persisted on alloc failure")
	}

	// Area oid was released.
	found := false
	for _, id := range alloc.released {
		if id == 1_000_001 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected area oid 1000001 to be released, released=%v", alloc.released)
	}
}

// -----------------------------------------------------------------------------
// GetByOwner
// -----------------------------------------------------------------------------

// TestGetByOwnerReturnsOnlyOwnersDoors seeds two owners' doors and asserts that
// GetByOwner returns only the requested owner's door(s).
func TestGetByOwnerReturnsOnlyOwnersDoors(t *testing.T) {
	res := fakeResolver{}
	alloc := &counterAllocator{next: 1}
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, res, alloc, em)

	f := field.NewBuilder(1, 2, 100000000).Build()
	townMapId := _map.Id(104000000)
	seedDoor(t, ctx, ten, 600_001, 600_002, 11, f, townMapId)
	seedDoor(t, ctx, ten, 610_001, 610_002, 22, f, townMapId)

	got, err := p.GetByOwner(11)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(got) != 1 || got[0].OwnerCharacterId() != 11 || got[0].AreaDoorId() != 600_001 {
		t.Fatalf("expected only owner 11's door 600001, got %+v", got)
	}

	other, err := p.GetByOwner(22)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(other) != 1 || other[0].OwnerCharacterId() != 22 || other[0].AreaDoorId() != 610_001 {
		t.Fatalf("expected only owner 22's door 610001, got %+v", other)
	}

	// Unknown owner returns no doors.
	none, err := p.GetByOwner(33)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no doors for unknown owner, got %+v", none)
	}
}

// -----------------------------------------------------------------------------
// RemoveByOwner
// -----------------------------------------------------------------------------

func TestRemoveByOwnerEmitsRemovedAndIsIdempotent(t *testing.T) {
	res := fakeResolver{}
	alloc := &counterAllocator{next: 1}
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, res, alloc, em)

	f := field.NewBuilder(1, 2, 100000000).Build()
	m := NewBuilder().
		SetAreaDoorId(700_001).SetTownDoorId(700_002).
		SetOwnerCharacterId(99).SetPartyId(0).SetField(f).
		SetTownMapId(_map.Id(104000000)).Build()
	if err := GetRegistry().Put(ctx, ten, m); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := p.RemoveByOwner(99, RemoveReasonLogout); err != nil {
		t.Fatalf("RemoveByOwner: %v", err)
	}
	if len(em.types) != 1 || em.types[0] != EventDoorStatusRemoved {
		t.Fatalf("expected [REMOVED], got %v", em.types)
	}
	if _, gerr := GetRegistry().Get(ctx, ten, 700_001); gerr == nil {
		t.Fatalf("door should be removed")
	}

	// Idempotent: second call removes nothing, emits nothing, no error.
	em2 := len(em.types)
	if err := p.RemoveByOwner(99, RemoveReasonLogout); err != nil {
		t.Fatalf("idempotent RemoveByOwner: %v", err)
	}
	if len(em.types) != em2 {
		t.Fatalf("second RemoveByOwner should emit nothing, types=%v", em.types)
	}
}

// -----------------------------------------------------------------------------
// RemoveByOwnerIfLeftField
// -----------------------------------------------------------------------------

// seedDoor is a test helper that places a door for ownerCharacterId in the
// registry using the given field/town/oid values.  It returns the seeded Model.
func seedDoor(t *testing.T, ctx context.Context, ten tenant.Model, areaDoorId, townDoorId uint32, ownerCharacterId character.Id, f field.Model, townMapId _map.Id) Model {
	t.Helper()
	m := NewBuilder().
		SetAreaDoorId(areaDoorId).SetTownDoorId(townDoorId).
		SetOwnerCharacterId(ownerCharacterId).SetPartyId(0).
		SetField(f).SetTownMapId(townMapId).SetSlot(0).Build()
	if err := GetRegistry().Put(ctx, ten, m); err != nil {
		t.Fatalf("seedDoor Put: %v", err)
	}
	return m
}

// decodeReason unpacks the Reason field from a REMOVED event body captured by
// fakeEmit.  It deserialises the raw kafka.Message value produced for a given
// index in the emit capture slice.
//
// Because fakeEmit only stores the outer "type" string we cannot go back to the
// raw bytes via em alone, so this helper re-emits through a thin capture that
// stores the full message bytes.
//
// Usage: pass a *reasonCapture as the emitter and call Reason(i).
type reasonCapture struct {
	messages [][]byte
}

func (rc *reasonCapture) emit(topic string, p model.Provider[[]kafka.Message]) error {
	msgs, err := p()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		rc.messages = append(rc.messages, msg.Value)
	}
	return nil
}

func (rc *reasonCapture) Reason(i int) string {
	if i >= len(rc.messages) {
		return ""
	}
	var env struct {
		Type string `json:"type"`
		Body struct {
			Reason string `json:"reason"`
		} `json:"body"`
	}
	_ = json.Unmarshal(rc.messages[i], &env)
	return env.Body.Reason
}

// TestRemoveByOwnerIfLeftField_LeavesToUnrelatedMap verifies that when the owner
// moves to a map that is neither the door's source field nor its town map the
// door is removed and a REMOVED/LEFT_FIELD event is emitted (FR-6.2).
func TestRemoveByOwnerIfLeftField_LeavesToUnrelatedMap(t *testing.T) {
	alloc := &counterAllocator{next: 900_001}
	rc := &reasonCapture{}
	ten, ctx := newTestTenant()
	GetRegistry().Clear(ctx)
	p := &ProcessorImpl{
		l:     logrus.New(),
		ctx:   ctx,
		t:     ten,
		emit:  rc.emit,
		res:   fakeResolver{},
		alloc: alloc,
	}

	srcField := field.NewBuilder(1, 2, 100000000).Build()
	townMapId := _map.Id(104000000)
	seedDoor(t, ctx, ten, 900_001, 900_002, 55, srcField, townMapId)

	// Move to an unrelated map — should trigger removal.
	unrelatedField := field.NewBuilder(1, 2, 999000000).Build()
	if err := p.RemoveByOwnerIfLeftField(55, unrelatedField); err != nil {
		t.Fatalf("RemoveByOwnerIfLeftField: %v", err)
	}

	// Exactly one REMOVED event with reason LEFT_FIELD.
	if len(rc.messages) != 1 {
		t.Fatalf("expected 1 emitted message, got %d", len(rc.messages))
	}
	if rc.Reason(0) != RemoveReasonLeftField {
		t.Fatalf("expected reason %s, got %s", RemoveReasonLeftField, rc.Reason(0))
	}

	// Door is gone from the registry.
	if _, err := GetRegistry().Get(ctx, ten, 900_001); err == nil {
		t.Fatalf("door 900001 should be removed from registry")
	}
	byOwner, err := GetRegistry().GetByOwner(ctx, ten, 55)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(byOwner) != 0 {
		t.Fatalf("expected no doors for owner after removal, got %d", len(byOwner))
	}

	// Both oids were released.
	released := map[uint32]bool{}
	for _, id := range alloc.released {
		released[id] = true
	}
	if !released[900_001] || !released[900_002] {
		t.Fatalf("expected oids 900001 and 900002 to be released, released=%v", alloc.released)
	}
}

// TestRemoveByOwnerIfLeftField_StaysInSourceField verifies that a field-change
// event to the exact same field as the door's source is a no-op: the door stays
// and no event is emitted (FR-6.2 stay-in-field guard).
func TestRemoveByOwnerIfLeftField_StaysInSourceField(t *testing.T) {
	alloc := &counterAllocator{next: 910_001}
	rc := &reasonCapture{}
	ten, ctx := newTestTenant()
	GetRegistry().Clear(ctx)
	p := &ProcessorImpl{
		l:     logrus.New(),
		ctx:   ctx,
		t:     ten,
		emit:  rc.emit,
		res:   fakeResolver{},
		alloc: alloc,
	}

	srcField := field.NewBuilder(1, 2, 100000000).Build()
	townMapId := _map.Id(104000000)
	seedDoor(t, ctx, ten, 910_001, 910_002, 56, srcField, townMapId)

	// newField == srcField → no removal.
	if err := p.RemoveByOwnerIfLeftField(56, srcField); err != nil {
		t.Fatalf("RemoveByOwnerIfLeftField: %v", err)
	}

	// No event emitted.
	if len(rc.messages) != 0 {
		t.Fatalf("expected no emit when staying in source field, got %d message(s)", len(rc.messages))
	}

	// Door still in registry.
	if _, err := GetRegistry().Get(ctx, ten, 910_001); err != nil {
		t.Fatalf("door should still be in registry: %v", err)
	}
}

// TestRemoveByOwnerIfLeftField_WarpIntoTown verifies that warping into the
// door's town map is treated as a valid transit (not abandonment) and the door
// is preserved with no event emitted (FR-6.2 into-town guard, design §5.3).
func TestRemoveByOwnerIfLeftField_WarpIntoTown(t *testing.T) {
	alloc := &counterAllocator{next: 920_001}
	rc := &reasonCapture{}
	ten, ctx := newTestTenant()
	GetRegistry().Clear(ctx)
	p := &ProcessorImpl{
		l:     logrus.New(),
		ctx:   ctx,
		t:     ten,
		emit:  rc.emit,
		res:   fakeResolver{},
		alloc: alloc,
	}

	srcField := field.NewBuilder(1, 2, 100000000).Build()
	townMapId := _map.Id(104000000)
	seedDoor(t, ctx, ten, 920_001, 920_002, 57, srcField, townMapId)

	// newField has the town map id → no removal.
	townField := field.NewBuilder(1, 2, townMapId).Build()
	if err := p.RemoveByOwnerIfLeftField(57, townField); err != nil {
		t.Fatalf("RemoveByOwnerIfLeftField: %v", err)
	}

	// No event emitted.
	if len(rc.messages) != 0 {
		t.Fatalf("expected no emit when warping into town, got %d message(s)", len(rc.messages))
	}

	// Door still in registry.
	if _, err := GetRegistry().Get(ctx, ten, 920_001); err != nil {
		t.Fatalf("door should still be in registry: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Reslot
// -----------------------------------------------------------------------------

func TestReslotEmitsSlotChangedAndNoOpsWhenUnchanged(t *testing.T) {
	res := fakeResolver{}
	alloc := &counterAllocator{next: 1}
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, res, alloc, em)

	f := field.NewBuilder(1, 2, 100000000).Build()
	m := NewBuilder().
		SetAreaDoorId(800_001).SetTownDoorId(800_002).
		SetOwnerCharacterId(77).SetPartyId(123).SetField(f).
		SetTownMapId(_map.Id(104000000)).SetSlot(0).SetTownPortalId(0x80).Build()
	if err := GetRegistry().Put(ctx, ten, m); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// No-op when slot unchanged.
	if err := p.Reslot(800_001, 0, 0x80, 0, 0); err != nil {
		t.Fatalf("Reslot no-op: %v", err)
	}
	if len(em.types) != 0 {
		t.Fatalf("expected no emit on unchanged slot, got %v", em.types)
	}

	// Change slot → SLOT_CHANGED emitted, persisted state updated.
	if err := p.Reslot(800_001, 2, 0x82, 50, 60); err != nil {
		t.Fatalf("Reslot change: %v", err)
	}
	if len(em.types) != 1 || em.types[0] != EventDoorStatusSlotChanged {
		t.Fatalf("expected [SLOT_CHANGED], got %v", em.types)
	}
	got, err := GetRegistry().Get(ctx, ten, 800_001)
	if err != nil {
		t.Fatalf("Get after reslot: %v", err)
	}
	if got.Slot() != 2 || got.TownPortalId() != 0x82 || got.TownX() != 50 || got.TownY() != 60 {
		t.Fatalf("reslot not persisted: slot=%d portal=%d x=%d y=%d", got.Slot(), got.TownPortalId(), got.TownX(), got.TownY())
	}
}

// TestLeavePartyDoorRemovesFromPartyThenRekeysSolo pins the party-leave fix: a
// departed member's door is REMOVED while still party-scoped (broadcast to the
// remaining members) and then re-CREATED as a solo door (party 0, slot 0). This
// is what stops the leaver's door lingering on a remaining member's client and
// being dragged onto slot 0 by a stale party-scoped reslot.
func TestLeavePartyDoorRemovesFromPartyThenRekeysSolo(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)

	f := field.NewBuilder(1, 2, 240011000).Build()
	// Leaver's door: party 123, slot 1 (a non-leader's party slot).
	m := NewBuilder().
		SetAreaDoorId(900_001).SetTownDoorId(900_002).
		SetOwnerCharacterId(5).SetPartyId(123).SetField(f).
		SetTownMapId(_map.Id(240000000)).SetSlot(1).SetTownPortalId(0x81).
		SetTownX(-85).SetTownY(531).Build()
	if err := GetRegistry().Put(ctx, ten, m); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Slot-0 town portal for the solo re-key.
	portals := []TownPortal{{X: 10, Y: 20}, {X: -85, Y: 531}}
	p.LeavePartyDoor(123, 5, func(_ _map.Id) []TownPortal { return portals })

	if len(em.types) != 2 || em.types[0] != EventDoorStatusRemoved || em.types[1] != EventDoorStatusCreated {
		t.Fatalf("expected [REMOVED, CREATED], got %v", em.types)
	}
	decode := func(b []byte) (partyId, forCh uint32) {
		var env struct {
			PartyId        uint32 `json:"partyId"`
			ForCharacterId uint32 `json:"forCharacterId"`
		}
		_ = json.Unmarshal(b, &env)
		return env.PartyId, env.ForCharacterId
	}
	// REMOVED is still party-scoped and broadcast (forCharacterId 0) so it reaches
	// — and clears the town-portal slot for — the remaining members.
	if pid, fc := decode(em.values[0]); pid != 123 || fc != 0 {
		t.Fatalf("REMOVED should be party-scoped broadcast: partyId=%d forCharacterId=%d", pid, fc)
	}
	// CREATED is solo (party 0) so it reaches only the owner.
	if pid, fc := decode(em.values[1]); pid != 0 || fc != 0 {
		t.Fatalf("CREATED should be solo: partyId=%d forCharacterId=%d", pid, fc)
	}
	// Persisted door is now solo at slot 0 with the slot-0 town portal.
	got, err := GetRegistry().Get(ctx, ten, 900_001)
	if err != nil {
		t.Fatalf("Get after leave: %v", err)
	}
	if got.PartyId() != 0 || got.Slot() != 0 || got.TownPortalId() != 0x80 || got.TownX() != 10 || got.TownY() != 20 {
		t.Fatalf("not re-keyed to solo: party=%d slot=%d portal=%d x=%d y=%d",
			got.PartyId(), got.Slot(), got.TownPortalId(), got.TownX(), got.TownY())
	}
}
