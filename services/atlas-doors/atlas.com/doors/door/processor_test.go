package door

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
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
	members     []uint32
	durationMs  int32
	resolveErr  error
}

func (f fakeResolver) PartyIdFor(_ context.Context, _ uint32) (uint32, error) {
	return f.partyId, nil
}

func (f fakeResolver) ResolveSpawn(_ context.Context, _ field.Model, ownerCharacterId, partyId, _ uint32, _ byte) (spawnPlan, error) {
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

// fakeEmit captures (topic, decoded event Type) for each emitted message.
type fakeEmit struct {
	topics []string
	types  []string
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
