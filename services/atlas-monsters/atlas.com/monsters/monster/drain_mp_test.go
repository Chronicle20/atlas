package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// TestDrainMp_HappyPath_EmitsMpChanged verifies that DrainMp on a non-boss
// monster with MaxMp=1000, Mp=1000 deducts 100 MP and emits a single
// MP_CHANGED event with Reason=MP_EATER, Amount=100, MonsterMpAfter=900.
func TestDrainMp_HappyPath_EmitsMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	// Stub information lookup to return a non-boss monster.
	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, uniqueId, 42, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}

	// Verify registry state.
	got, getErr := r.GetMonster(ten, uniqueId)
	if getErr != nil {
		t.Fatalf("GetMonster: %v", getErr)
	}
	if got.Mp() != 900 {
		t.Errorf("Mp after DrainMp = %d, want 900", got.Mp())
	}

	// Verify exactly one MP_CHANGED event.
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(*events), *events)
	}
	ev := (*events)[0]
	if ev.Type != EventMonsterStatusMpChanged {
		t.Errorf("event type = %q, want %q", ev.Type, EventMonsterStatusMpChanged)
	}
	if ev.Topic != EnvEventTopicMonsterStatus {
		t.Errorf("event topic = %q, want %q", ev.Topic, EnvEventTopicMonsterStatus)
	}

	var body statusEventMpChangedBody
	if err := json.Unmarshal(ev.Body, &body); err != nil {
		t.Fatalf("decode MP_CHANGED body: %v", err)
	}
	if body.Reason != MpChangeReasonMpEater {
		t.Errorf("body.Reason = %q, want %q", body.Reason, MpChangeReasonMpEater)
	}
	if body.Amount != 100 {
		t.Errorf("body.Amount = %d, want 100", body.Amount)
	}
	if body.MonsterMpAfter != 900 {
		t.Errorf("body.MonsterMpAfter = %d, want 900", body.MonsterMpAfter)
	}
	if body.CharacterId != 42 {
		t.Errorf("body.CharacterId = %d, want 42", body.CharacterId)
	}
	if body.SkillId != 2300000 {
		t.Errorf("body.SkillId = %d, want 2300000", body.SkillId)
	}
}

// TestDrainMp_ClampsAtZero verifies that when the monster has Mp=50 and the
// request is 200, the registry mutation clamps at 0 (Mp→0) but the emitted
// event still reports Amount=requestedAmount=200 — the channel refunds the
// caster the channel-computed amount regardless of the monster's actual
// MP loss.
func TestDrainMp_ClampsAtZero(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()
	// Manually reduce MP to 50 by deducting 950.
	if _, err := r.DeductMp(ten, uniqueId, 950); err != nil {
		t.Fatalf("seed DeductMp: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, uniqueId, 1, 2300000, 200)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}

	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 0 {
		t.Errorf("Mp after clamp-at-zero DrainMp = %d, want 0", got.Mp())
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}
	var body statusEventMpChangedBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Amount != 200 {
		t.Errorf("body.Amount = %d, want 200 (requested amount; not the clamped actual)", body.Amount)
	}
	if body.MonsterMpAfter != 0 {
		t.Errorf("body.MonsterMpAfter = %d, want 0", body.MonsterMpAfter)
	}
}

// TestDrainMp_SkipsZeroMaxMp verifies that DrainMp emits nothing when
// MaxMp == 0 (monster was created with zero max MP).
func TestDrainMp_SkipsZeroMaxMp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	// Create with mp=0 → MaxMp==0.
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 0)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, uniqueId, 1, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for MaxMp=0 monster, got %d: %v", len(*events), *events)
	}
}

// TestDrainMp_DryMonsterStillEmits verifies that DrainMp emits MP_CHANGED
// even when the monster's current Mp is already 0 (e.g., a prior drain
// emptied it). The channel still refunds the caster and plays the visual
// — Cosmic does not gate the proc effect on the monster's remaining MP.
// The registry remains at Mp=0 (no further deduct).
func TestDrainMp_DryMonsterStillEmits(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()
	// Drain all MP first.
	if _, err := r.DeductMp(ten, uniqueId, 1000); err != nil {
		t.Fatalf("seed DeductMp: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, uniqueId, 1, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event (cosmetic emit), got %d", len(*events))
	}
	var body statusEventMpChangedBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Amount != 100 {
		t.Errorf("body.Amount = %d, want 100", body.Amount)
	}
	if body.MonsterMpAfter != 0 {
		t.Errorf("body.MonsterMpAfter = %d, want 0", body.MonsterMpAfter)
	}
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 0 {
		t.Errorf("Mp should remain 0; got %d", got.Mp())
	}
}

// TestDrainMp_SkipsZeroRequest verifies that DrainMp with requestedAmount=0
// emits nothing and leaves Mp unchanged.
func TestDrainMp_SkipsZeroRequest(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, uniqueId, 1, 2300000, 0)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for requestedAmount=0, got %d", len(*events))
	}
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 1000 {
		t.Errorf("Mp after zero-request DrainMp = %d, want 1000 (unchanged)", got.Mp())
	}
}

// TestDrainMp_MissingMonster verifies that DrainMp emits MP_CHANGED with a
// synthetic post-mortem snapshot when the uniqueId is not present in the
// registry. Real-world cause: the monster was one-shot killed by the same
// player attack — DAMAGE and DRAIN_MP are produced from a single
// processAttack and partitioned by uniqueId, so DAMAGE processes (and
// destroys) before DRAIN_MP arrives. Cosmic still plays the visual and
// refunds the caster on kill shots.
func TestDrainMp_MissingMonster(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, 99999999, 1, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error for missing monster: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event (post-mortem cosmetic emit), got %d", len(*events))
	}
	var body statusEventMpChangedBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Reason != MpChangeReasonMpEater {
		t.Errorf("body.Reason = %q, want %q", body.Reason, MpChangeReasonMpEater)
	}
	if body.Amount != 100 {
		t.Errorf("body.Amount = %d, want 100 (channel-computed refund amount)", body.Amount)
	}
	if body.MonsterMpAfter != 0 {
		t.Errorf("body.MonsterMpAfter = %d, want 0 (post-mortem)", body.MonsterMpAfter)
	}
	if body.CharacterId != 1 {
		t.Errorf("body.CharacterId = %d, want 1", body.CharacterId)
	}
	if body.SkillId != 2300000 {
		t.Errorf("body.SkillId = %d, want 2300000", body.SkillId)
	}
}

// TestDrainMp_SkipsBoss verifies that DrainMp emits nothing and leaves Mp
// unchanged when the information lookup indicates the monster is a boss.
func TestDrainMp_SkipsBoss(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(true).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 8800000, 0, 0, 0, 5, 0, 50000, 3000)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	err := p.DrainMp(f, uniqueId, 1, 2300000, 500)
	if err != nil {
		t.Fatalf("DrainMp returned error for boss: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for boss monster, got %d", len(*events))
	}
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 3000 {
		t.Errorf("Mp after boss DrainMp = %d, want 3000 (unchanged)", got.Mp())
	}
}

