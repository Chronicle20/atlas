package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestKill_NonBoss_KilledAndRemoved — a Mortal Blow kill on a non-boss at
// any HP emits DAMAGED then KILLED, removes the monster from the registry,
// and credits the full remaining HP to the killer in the damage summary
// (ApplyDamage clamps the MaxUint32 line to the HP actually removed).
func TestKill_NonBoss_KilledAndRemoved(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 100, "", "")
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (DAMAGED, KILLED), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0].Type = %q, want %q", (*events)[0].Type, EventMonsterStatusDamaged)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1].Type = %q, want %q", (*events)[1].Type, EventMonsterStatusKilled)
	}

	var body statusEventKilledBody
	if err := json.Unmarshal((*events)[1].Body, &body); err != nil {
		t.Fatalf("decode KILLED body: %v", err)
	}
	if body.ActorId != 42 {
		t.Errorf("KILLED.ActorId = %d, want 42", body.ActorId)
	}
	if len(body.DamageEntries) != 1 {
		t.Fatalf("KILLED.DamageEntries = %v, want exactly 1 entry", body.DamageEntries)
	}
	if body.DamageEntries[0].CharacterId != 42 {
		t.Errorf("DamageEntries[0].CharacterId = %d, want 42", body.DamageEntries[0].CharacterId)
	}
	if body.DamageEntries[0].Damage != 5000 {
		t.Errorf("DamageEntries[0].Damage = %d, want 5000 (clamped to HP removed, not MaxUint32)", body.DamageEntries[0].Damage)
	}

	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Errorf("expected monster [%d] removed from registry after kill", uniqueId)
	}
}

// TestKill_Boss_Dropped — the authoritative boss guard: no events, monster
// untouched, regardless of what the channel decided.
func TestKill_Boss_Dropped(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewBuilder().SetBoss(true).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 8800000, 0, 0, 0, 5, 0, 50000, 3000, "", "")
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for boss kill attempt, got %d: %v", len(*events), *events)
	}
	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("boss must remain in registry: %v", err)
	}
	if got.Hp() != 50000 {
		t.Errorf("boss HP = %d, want 50000 (untouched)", got.Hp())
	}
}

// TestKill_InfoLookupError_DroppedFailClosed — if the boss lookup errors,
// the kill is dropped. This deliberately diverges from DrainMp's fail-open:
// losing a legitimate proc during an atlas-data hiccup is acceptable;
// killing a boss is not (FR-4).
func TestKill_InfoLookupError_DroppedFailClosed(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.Model{}, errors.New("atlas-data unavailable")
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 100, "", "")
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events on info-lookup error (fail-closed), got %d: %v", len(*events), *events)
	}
	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("monster must remain in registry: %v", err)
	}
	if got.Hp() != 5000 {
		t.Errorf("HP = %d, want 5000 (untouched)", got.Hp())
	}
}

// TestKill_MissingMonster_NoOp — the triggering attack already killed the
// monster (DAMAGE and KILL share a partition; DAMAGE processed first) or
// it despawned. Nothing to do, nothing emitted.
func TestKill_MissingMonster_NoOp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(99999999, 42)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for missing monster, got %d: %v", len(*events), *events)
	}
}

// TestKill_DeadMonster_NoOp — a registry entry at HP 0 (killed but not yet
// removed) is dropped without events.
func TestKill_DeadMonster_NoOp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 1, 50, "", "")
	uniqueId := m.UniqueId()
	// Kill it directly via the registry (no emit) so HP=0 but it remains present.
	if _, err := r.ApplyDamage(ten, 1, 999, uniqueId, 1); err != nil {
		t.Fatalf("seed ApplyDamage: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Kill(uniqueId, 42)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for dead monster, got %d: %v", len(*events), *events)
	}
}
