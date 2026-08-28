package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/character/skill"
	"atlas-channel/character/snapshot"
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	"atlas-channel/monster"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	invconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestComputeReflect_InsideRange_ReturnsClampedReflectedDamage exercises the
// happy path: attacker inside the bounding box, total damage reflected at
// the configured percent, and clamped if it would exceed MaxDamage.
func TestComputeReflect_InsideRange_ReturnsClampedReflectedDamage(t *testing.T) {
	info := monster.ReflectInfo{
		Kind:      monster2.ReflectKindPhysical,
		Percent:   30,
		LtX:       -100,
		LtY:       -100,
		RbX:       100,
		RbY:       100,
		MaxDamage: 9999,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	// attacker at (50, 0); monster at (0, 0); dx=50 dy=0, well inside.
	reflected, within := computeReflect([]int32{1000}, info, 50, 0, 0, 0)
	if !within {
		t.Fatalf("expected withinRange=true")
	}
	if reflected != 300 {
		t.Fatalf("reflected = %d, want 300", reflected)
	}
}

// TestComputeReflect_OutsideRange_ReturnsZero confirms attacker outside the
// monster's reflect window receives no reflected damage.
func TestComputeReflect_OutsideRange_ReturnsZero(t *testing.T) {
	info := monster.ReflectInfo{
		Kind:      monster2.ReflectKindPhysical,
		Percent:   30,
		LtX:       -100,
		LtY:       -100,
		RbX:       100,
		RbY:       100,
		MaxDamage: 9999,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	// attacker at (200, 0); monster at (0, 0); dx=200, > RbX=100.
	reflected, within := computeReflect([]int32{1000}, info, 200, 0, 0, 0)
	if within {
		t.Fatalf("expected withinRange=false")
	}
	if reflected != 0 {
		t.Fatalf("reflected = %d, want 0", reflected)
	}
}

// TestComputeReflect_SumsAllDamages confirms multi-line damage is summed
// before applying the reflect percent.
func TestComputeReflect_SumsAllDamages(t *testing.T) {
	info := monster.ReflectInfo{
		Kind:      monster2.ReflectKindPhysical,
		Percent:   50,
		LtX:       -100,
		LtY:       -100,
		RbX:       100,
		RbY:       100,
		MaxDamage: 99999,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	// total = 100+200+400 = 700; reflected = 700*50/100 = 350.
	reflected, within := computeReflect([]int32{100, 200, 400}, info, 0, 0, 0, 0)
	if !within {
		t.Fatalf("expected withinRange=true")
	}
	if reflected != 350 {
		t.Fatalf("reflected = %d, want 350", reflected)
	}
}

// TestComputeReflect_ClampsToMaxDamage confirms the reflected value is
// clamped to MaxDamage when the percent calculation would exceed it.
func TestComputeReflect_ClampsToMaxDamage(t *testing.T) {
	info := monster.ReflectInfo{
		Kind:      monster2.ReflectKindPhysical,
		Percent:   50,
		LtX:       -100,
		LtY:       -100,
		RbX:       100,
		RbY:       100,
		MaxDamage: 100,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	// reflected = 1000*50/100 = 500, but MaxDamage=100 clamps to 100.
	reflected, within := computeReflect([]int32{1000}, info, 0, 0, 0, 0)
	if !within {
		t.Fatalf("expected withinRange=true")
	}
	if reflected != 100 {
		t.Fatalf("reflected = %d, want 100 (clamped)", reflected)
	}
}

// TestComputeReflect_BoundaryEdgesAreInclusive — points exactly on the
// LtX/LtY/RbX/RbY edges count as inside.
func TestComputeReflect_BoundaryEdgesAreInclusive(t *testing.T) {
	info := monster.ReflectInfo{
		Kind:      monster2.ReflectKindPhysical,
		Percent:   10,
		LtX:       -100,
		LtY:       -100,
		RbX:       100,
		RbY:       100,
		MaxDamage: 9999,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	// Attacker at exactly RbX, RbY.
	if _, within := computeReflect([]int32{500}, info, 100, 100, 0, 0); !within {
		t.Fatalf("expected withinRange=true at RbX/RbY corner")
	}
	// Attacker at exactly LtX, LtY.
	if _, within := computeReflect([]int32{500}, info, -100, -100, 0, 0); !within {
		t.Fatalf("expected withinRange=true at LtX/LtY corner")
	}
}

// TestComputeReflect_KindMismatch is exercised at the orchestration layer
// (the handler chooses kind via mirror.GetReflect). The pure helper is
// kind-agnostic, so we don't test mismatch here. This sentinel keeps the
// suite explicit about the test seam boundary.
func TestComputeReflect_KindMismatchHandledByCaller(t *testing.T) {
	// Intentionally a no-op: reflect kind matching is the caller's job.
	// The helper trusts that GetReflect already filtered by kind.
	_ = packetmodel.AttackTypeMagic
}

// TestReflectFlow_PhysicalInsideRange_EmitsReflectAndSkipsDamage exercises
// the full orchestration the handler performs for one damage entry:
// (1) AttackType -> kind, (2) StatusMirror.GetReflect, (3) bounding-box
// check, (4) reflect-or-damage decision. We don't drive the real handler
// (it requires sessions and REST clients) — the test composes the same
// pieces the handler composes. Failing in any of those steps would still
// surface here.
func TestReflectFlow_PhysicalInsideRange_EmitsReflect(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	mirror := monster.GetStatusMirror()

	uniqueId := uint32(700001)
	mirror.OnApplied(tm, uniqueId, monster.StatusEffectAppliedBody{
		EffectId:         uuid.NewString(),
		Statuses:         map[string]int32{"WEAPON_REFLECT": 1},
		Duration:         60000,
		ReflectKind:      monster2.ReflectKindPhysical,
		ReflectPercent:   30,
		ReflectLtX:       -100,
		ReflectLtY:       -100,
		ReflectRbX:       100,
		ReflectRbY:       100,
		ReflectMaxDamage: 9999,
	}, time.Now())
	t.Cleanup(func() { mirror.OnMonsterGone(tm, uniqueId) })

	kind := attackKindFromAttackType(packetmodel.AttackTypeMelee)
	if kind != monster2.ReflectKindPhysical {
		t.Fatalf("kind = %q, want PHYSICAL", kind)
	}
	info, ok := mirror.GetReflect(tm, uniqueId, kind)
	if !ok {
		t.Fatalf("expected reflect info for monster %d kind %s", uniqueId, kind)
	}
	r, within := computeReflect([]int32{1000}, info /*charX*/, 50, 0 /*monX*/, 0, 0)
	if !within {
		t.Fatalf("expected within range")
	}
	if r != 300 {
		t.Fatalf("reflected = %d, want 300", r)
	}
}

// TestReflectFlow_MagicAttackOnPhysicalReflect_NoReflect verifies the
// attack kind / reflect kind mismatch suppresses the reflect.
func TestReflectFlow_MagicAttackOnPhysicalReflect_NoReflect(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	mirror := monster.GetStatusMirror()

	uniqueId := uint32(700002)
	mirror.OnApplied(tm, uniqueId, monster.StatusEffectAppliedBody{
		EffectId:         uuid.NewString(),
		Statuses:         map[string]int32{"WEAPON_REFLECT": 1},
		Duration:         60000,
		ReflectKind:      monster2.ReflectKindPhysical,
		ReflectPercent:   30,
		ReflectLtX:       -100,
		ReflectLtY:       -100,
		ReflectRbX:       100,
		ReflectRbY:       100,
		ReflectMaxDamage: 9999,
	}, time.Now())
	t.Cleanup(func() { mirror.OnMonsterGone(tm, uniqueId) })

	kind := attackKindFromAttackType(packetmodel.AttackTypeMagic)
	if _, ok := mirror.GetReflect(tm, uniqueId, kind); ok {
		t.Fatalf("expected no MAGICAL reflect on PHYSICAL-only monster")
	}
}

// TestReflectFlow_MagicAttackOnMagicalReflect_Reflects verifies the
// magical-on-magical path works symmetrically.
func TestReflectFlow_MagicAttackOnMagicalReflect_Reflects(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	mirror := monster.GetStatusMirror()

	uniqueId := uint32(700003)
	mirror.OnApplied(tm, uniqueId, monster.StatusEffectAppliedBody{
		EffectId:         uuid.NewString(),
		Statuses:         map[string]int32{"MAGIC_REFLECT": 1},
		Duration:         60000,
		ReflectKind:      monster2.ReflectKindMagical,
		ReflectPercent:   25,
		ReflectLtX:       -200,
		ReflectLtY:       -200,
		ReflectRbX:       200,
		ReflectRbY:       200,
		ReflectMaxDamage: 9999,
	}, time.Now())
	t.Cleanup(func() { mirror.OnMonsterGone(tm, uniqueId) })

	kind := attackKindFromAttackType(packetmodel.AttackTypeMagic)
	info, ok := mirror.GetReflect(tm, uniqueId, kind)
	if !ok {
		t.Fatalf("expected MAGICAL reflect")
	}
	r, within := computeReflect([]int32{800}, info, 0, 0, 0, 0)
	if !within || r != 200 {
		t.Fatalf("reflected = %d within = %v, want 200 / true", r, within)
	}
}

// TestReflectFlow_AfterExpiry_NoReflect captures the risks-doc regression:
// once a reflect's ExpiresAt has passed, a stale entry must not still
// trigger reflect. The mirror filters wall-clock-expired entries inside
// GetReflect, so the orchestrator naturally lands on the no-reflect path.
func TestReflectFlow_AfterExpiry_NoReflect(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	mirror := monster.GetStatusMirror()

	uniqueId := uint32(700004)
	// Apply with `now` two hours ago so the 60s duration has already lapsed.
	past := time.Now().Add(-2 * time.Hour)
	mirror.OnApplied(tm, uniqueId, monster.StatusEffectAppliedBody{
		EffectId:         uuid.NewString(),
		Statuses:         map[string]int32{"WEAPON_REFLECT": 1},
		Duration:         60000,
		ReflectKind:      monster2.ReflectKindPhysical,
		ReflectPercent:   30,
		ReflectLtX:       -100,
		ReflectLtY:       -100,
		ReflectRbX:       100,
		ReflectRbY:       100,
		ReflectMaxDamage: 9999,
	}, past)
	t.Cleanup(func() { mirror.OnMonsterGone(tm, uniqueId) })

	if _, ok := mirror.GetReflect(tm, uniqueId, monster2.ReflectKindPhysical); ok {
		t.Fatalf("expected expired reflect to be skipped by GetReflect")
	}
}

// TestSnapshotVenomDamagePerTick_LowCoef pins the rounded value at the
// low end of the [0.1, 0.2) coefficient range. Luck=120, MAtk=200,
// coef=0.1 -> round(0.1 * 120 * 200) = 2400.
func TestSnapshotVenomDamagePerTick_LowCoef(t *testing.T) {
	if got := snapshotVenomDamagePerTick(120, 200, 0.1); got != 2400 {
		t.Fatalf("snapshotVenomDamagePerTick(120, 200, 0.1) = %d, want 2400", got)
	}
}

// TestSnapshotVenomDamagePerTick_HighCoef pins the rounded value at the
// high end of the [0.1, 0.2) coefficient range. coef=0.2 -> 4800.
func TestSnapshotVenomDamagePerTick_HighCoef(t *testing.T) {
	if got := snapshotVenomDamagePerTick(120, 200, 0.2); got != 4800 {
		t.Fatalf("snapshotVenomDamagePerTick(120, 200, 0.2) = %d, want 4800", got)
	}
}

// TestSnapshotVenomDamagePerTick_Mid pins a midpoint value to confirm
// rounding behaviour matches math.Round (banker's rounding NOT used).
func TestSnapshotVenomDamagePerTick_Mid(t *testing.T) {
	if got := snapshotVenomDamagePerTick(120, 200, 0.15); got != 3600 {
		t.Fatalf("snapshotVenomDamagePerTick(120, 200, 0.15) = %d, want 3600", got)
	}
}

// TestAttackKindFromAttackType maps each AttackType to the reflect kind
// the handler will look up in the StatusMirror.
func TestAttackKindFromAttackType(t *testing.T) {
	cases := []struct {
		name string
		at   packetmodel.AttackType
		want string
	}{
		{"melee -> physical", packetmodel.AttackTypeMelee, monster2.ReflectKindPhysical},
		{"ranged -> physical", packetmodel.AttackTypeRanged, monster2.ReflectKindPhysical},
		{"magic -> magical", packetmodel.AttackTypeMagic, monster2.ReflectKindMagical},
		{"energy -> none", packetmodel.AttackTypeEnergy, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := attackKindFromAttackType(tc.at); got != tc.want {
				t.Fatalf("attackKindFromAttackType(%v) = %q, want %q", tc.at, got, tc.want)
			}
		})
	}
}

// TestBeaconTargetMonsterId exercises the beacon lock-target selection: skip
// zero ids, skip monsters absent from the field registry, and let the LAST
// remaining valid entry win (Cosmic per-monster loop order; WZ has no
// mobCount for 5211006/5220011 so multi-entry only arises from malformed
// packets).
func TestBeaconTargetMonsterId(t *testing.T) {
	existsAll := func(uint32) bool { return true }

	// Whiff: no entries.
	if _, ok := beaconTargetMonsterId(nil, existsAll); ok {
		t.Fatal("whiff must yield no target")
	}
	// Zero monster ids are skipped.
	if _, ok := beaconTargetMonsterId([]uint32{0, 0}, existsAll); ok {
		t.Fatal("zero ids must yield no target")
	}
	// Single hit.
	id, ok := beaconTargetMonsterId([]uint32{1000001}, existsAll)
	if !ok || id != 1000001 {
		t.Fatalf("single hit: got %d ok=%v", id, ok)
	}
	// Multi-entry: LAST valid entry wins (Cosmic per-monster loop order,
	// design.md §5.2; WZ has no mobCount so this is a defensive rule).
	id, ok = beaconTargetMonsterId([]uint32{1000001, 1000002}, existsAll)
	if !ok || id != 1000002 {
		t.Fatalf("last-wins: got %d ok=%v", id, ok)
	}
	// Monster missing from the field registry is skipped.
	id, ok = beaconTargetMonsterId([]uint32{1000001, 1000002}, func(m uint32) bool { return m == 1000001 })
	if !ok || id != 1000001 {
		t.Fatalf("missing-monster skip: got %d ok=%v", id, ok)
	}
}

// TestBeaconTryApply exercises the beacon apply gate: non-beacon skills and
// whiffs emit nothing, a hit emits CANCEL_BY_TYPES then APPLY in that order,
// and a cancel failure prevents the apply from running (FR-1.4/FR-1.5/FR-1.6).
func TestBeaconTryApply(t *testing.T) {
	mkAttack := func(skillId uint32, monsterIds ...uint32) packetmodel.AttackInfo {
		ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged)
		ai.SetSkillId(skillId)
		for _, m := range monsterIds {
			di := packetmodel.NewDamageInfo(1)
			di.SetMonsterId(m)
			ai.AddDamageInfo(*di)
		}
		return *ai
	}

	type call struct {
		kind  string
		mobId int32
	}
	l := logrus.New()
	l.SetOutput(io.Discard)
	record := func(calls *[]call) beaconApplyDeps {
		return beaconApplyDeps{
			monsterExists: func(uint32) bool { return true },
			cancelByTypes: func(types []string) error {
				*calls = append(*calls, call{kind: "cancel"})
				return nil
			},
			applyNoExpiry: func(sourceId int32, level byte, mobId int32) error {
				*calls = append(*calls, call{kind: "apply", mobId: mobId})
				return nil
			},
		}
	}

	// Non-beacon skill: nothing emitted.
	var calls []call
	beaconTryApply(l, mkAttack(4001344, 1000001), 1, field.Model{}, 42, record(&calls))
	if len(calls) != 0 {
		t.Fatalf("non-beacon skill must emit nothing, got %v", calls)
	}

	// Whiff: nothing emitted, prior lock untouched (FR-1.5).
	calls = nil
	beaconTryApply(l, mkAttack(uint32(skill3.OutlawHomingBeaconId)), 1, field.Model{}, 42, record(&calls))
	if len(calls) != 0 {
		t.Fatalf("whiff must emit nothing, got %v", calls)
	}

	// Hit: CANCEL_BY_TYPES then APPLY, in that order (FR-1.4).
	calls = nil
	beaconTryApply(l, mkAttack(uint32(skill3.CorsairBullseyeId), 1000001), 10, field.Model{}, 42, record(&calls))
	if len(calls) != 2 || calls[0].kind != "cancel" || calls[1].kind != "apply" || calls[1].mobId != 1000001 {
		t.Fatalf("hit must emit cancel then apply(mobId): got %v", calls)
	}

	// Cancel failure: apply is not attempted (lock state stays consistent).
	calls = nil
	deps := record(&calls)
	deps.cancelByTypes = func([]string) error { return errors.New("kafka down") }
	beaconTryApply(l, mkAttack(uint32(skill3.OutlawHomingBeaconId), 1000001), 1, field.Model{}, 42, deps)
	for _, c := range calls {
		if c.kind == "apply" {
			t.Fatal("apply must not run after cancel failure")
		}
	}
}

// newHandlerTestInventory builds a minimal inventory (one consumable asset)
// for handler-package fixtures, matching the shape of the snapshot
// package's own testInventory helper (character/snapshot/registry_test.go).
func newHandlerTestInventory(t *testing.T, characterId uint32) (inventory.Model, uuid.UUID, asset.Model) {
	t.Helper()
	compId := uuid.New()
	a := asset.NewBuilderWithId(9001, compId, 2060000).SetSlot(2).SetQuantity(400).MustBuild()
	comp := compartment.NewBuilder(compId, characterId, invconst.TypeValueUse, 96).AddAsset(a).MustBuild()
	inv := inventory.NewBuilder(characterId).
		SetEquipable(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueEquip, 96).MustBuild()).
		SetConsumable(comp).
		SetSetup(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueSetup, 96).MustBuild()).
		SetEtc(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueETC, 96).MustBuild()).
		SetCash(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueCash, 96).MustBuild()).
		MustBuild()
	return inv, compId, a
}

// newRangedAttackInfoFixture builds a minimal ranged AttackInfo with one
// damage entry and SkillId=0, so the writer's mastery/skill-data lookup
// short-circuits (no owned skill match) while inventory-derived bullet data
// still runs through preComputeAttackValues.
func newRangedAttackInfoFixture(t *testing.T) packetmodel.AttackInfo {
	t.Helper()
	ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged)
	ai.AddDamageInfo(*packetmodel.NewDamageInfo(1).SetMonsterId(9001).SetDamages([]uint32{100}))
	return *ai
}

// FR-7.3: one monster resolve per damaged monster serves BOTH the reflect
// check and MP Eater, mirror hit or not.
func TestProcessAttack_MonsterResolveDeduped(t *testing.T) {
	// Drive processDamageInfoEntry twice for the same monster through deps
	// whose getMonster counts calls; then assert the memoized resolver
	// (buildMonsterResolver) collapses them.
	tm := testTenant(t)
	calls := 0
	prev := attackMonsterByIdFn
	attackMonsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		calls++
		f := field.NewBuilder(0, 1, 100000000).Build()
		return monster.NewBuilder(uniqueId, f, 100100).SetMp(50).SetMaxMp(80).SetX(10).SetY(20).Build()
	}
	defer func() { attackMonsterByIdFn = prev }()

	ctx := tenant.WithContext(context.Background(), tm)
	resolve := buildMonsterResolver(logrus.New(), ctx, tm)

	e1, err := resolve(4001)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if _, err = resolve(4001); err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolver must memoize per swing: %d REST calls", calls)
	}
	if e1.Mp != 50 || e1.MaxMp != 80 || e1.X != 10 || e1.Y != 20 {
		t.Fatalf("entry mismatch: %+v", e1)
	}
	// The fallback must have backfilled the mirror for the NEXT swing.
	if got, ok := monster.GetLiveMirror().Lookup(tm, 4001); !ok || got.Mp != 50 {
		t.Fatalf("fallback must backfill mirror: %+v ok=%v", got, ok)
	}
}

func TestBuildMonsterResolver_MirrorHitZeroRest(t *testing.T) {
	tm := testTenant(t)
	f := field.NewBuilder(0, 1, 100000000).Build()
	monster.GetLiveMirror().Put(tm, 4002, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 7, MaxMp: 9, X: 1, Y: 2})

	prev := attackMonsterByIdFn
	attackMonsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		t.Fatalf("mirror hit must not call REST")
		return monster.Model{}, nil
	}
	defer func() { attackMonsterByIdFn = prev }()

	ctx := tenant.WithContext(context.Background(), tm)
	resolve := buildMonsterResolver(logrus.New(), ctx, tm)
	e, err := resolve(4002)
	if err != nil || e.Mp != 7 || e.X != 1 {
		t.Fatalf("mirror entry mismatch: %+v err=%v", e, err)
	}
}

// TestBuildMonsterModelResolver_Memoizes pins R7: the full-Model resolver
// (drainTryHeal / pickPocketTryProc / mortalBlowDeps) is NOT backed by the
// mirror (it does not carry Hp/MaxHp), but it IS memoized per swing — two
// resolves for the same monster must still collapse to one REST call.
func TestBuildMonsterModelResolver_Memoizes(t *testing.T) {
	tm := testTenant(t)
	calls := 0
	prev := attackMonsterByIdFn
	attackMonsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		calls++
		f := field.NewBuilder(0, 1, 100000000).Build()
		return monster.NewBuilder(uniqueId, f, 100100).SetHp(10).SetMaxHp(100).Build()
	}
	defer func() { attackMonsterByIdFn = prev }()

	ctx := tenant.WithContext(context.Background(), tm)
	resolve := buildMonsterModelResolver(logrus.New(), ctx, tm)

	mo1, err := resolve(5001)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if _, err = resolve(5001); err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolver must memoize per swing: %d REST calls", calls)
	}
	if mo1.Hp() != 10 || mo1.MaxHp() != 100 {
		t.Fatalf("model mismatch: hp=%d maxHp=%d", mo1.Hp(), mo1.MaxHp())
	}
}

// FR-7.4 staleness window: a skill UPDATED event between two snapshot reads
// is reflected in the second read.
func TestSnapshotStalenessWindow_SkillLevelChangesBetweenAttacks(t *testing.T) {
	tm := testTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	v := snapshot.GetRegistry().View(tm, 91)
	core := character.NewBuilder().SetId(91).SetLevel(120).MustBuild()
	_ = snapshot.GetRegistry().BackfillCore(tm, 91, core, v.CoreGen)
	_ = snapshot.GetRegistry().BackfillSkills(tm, 91, []skill.Model{
		skill.NewModelBuilder(skill3.Id(3121004)).SetLevel(10).MustBuild(),
	}, v.SkillsGen)
	inv, _, _ := newHandlerTestInventory(t, 91) // package-local inventory fixture (same shape as snapshot tests)
	_ = snapshot.GetRegistry().BackfillInventory(tm, 91, inv, v.InvGen)
	_ = snapshot.GetRegistry().BackfillBuffs(tm, 91, nil, v.BuffsGen)

	sp := snapshot.NewProcessor(logrus.New(), ctx)
	m1, err := sp.Get(91)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if s, _ := m1.SkillById(skill3.Id(3121004)); s.Level() != 10 {
		t.Fatalf("first read level: %d", s.Level())
	}

	// Event lands between attacks.
	snapshot.GetRegistry().UpsertSkill(tm, 91,
		skill.NewModelBuilder(skill3.Id(3121004)).SetLevel(11).MustBuild())

	m2, err := sp.Get(91)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if s, _ := m2.SkillById(skill3.Id(3121004)); s.Level() != 11 {
		t.Fatalf("second attack must see the new level: %d", s.Level())
	}
}

// FR-4.6/FR-7.2: for the same logical state, the snapshot-composed model
// and the decorator-built model produce byte-identical broadcast bodies.
func TestWriterEquivalence_SnapshotComposedModel(t *testing.T) {
	tm := testTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	inv, _, _ := newHandlerTestInventory(t, 92)
	skills := []skill.Model{skill.NewModelBuilder(skill3.Id(3121004)).SetLevel(10).MustBuild()}
	base := character.NewBuilder().SetId(92).SetLevel(120).SetJobId(322).SetX(5).SetY(-5).MustBuild()

	// Path A: today's decorator order (GetById -> InventoryDecorator -> SkillModelDecorator).
	want := base.SetInventory(inv).SetSkills(skills)

	// Path B: snapshot composition.
	v := snapshot.GetRegistry().View(tm, 92)
	_ = snapshot.GetRegistry().BackfillCore(tm, 92, base, v.CoreGen)
	_ = snapshot.GetRegistry().BackfillSkills(tm, 92, skills, v.SkillsGen)
	_ = snapshot.GetRegistry().BackfillInventory(tm, 92, inv, v.InvGen)
	_ = snapshot.GetRegistry().BackfillBuffs(tm, 92, nil, v.BuffsGen)
	got, err := snapshot.NewProcessor(logrus.New(), ctx).Get(92)
	if err != nil {
		t.Fatalf("snapshot get: %v", err)
	}

	ai := newRangedAttackInfoFixture(t) // reuse/extend the package's existing AttackInfo fixture

	// packet.Encode = func(l, ctx) func(options map[string]interface{}) []byte
	// (libs/atlas-socket/packet/encoder.go:9). Use the packet-test context
	// helper the writer tests use (socket/writer/character_info_test.go:31,
	// pt "github.com/Chronicle20/atlas/libs/atlas-packet/test").
	ptCtx := pt.CreateContext("GMS", 83, 1)
	lb := logrus.New()
	opts := map[string]interface{}{}
	wantBytes := writer.CharacterAttackRangedBody(want, ai)(lb, ptCtx)(opts)
	gotBytes := writer.CharacterAttackRangedBody(got, ai)(lb, ptCtx)(opts)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Fatalf("broadcast bytes diverge:\nwant %x\ngot  %x", wantBytes, gotBytes)
	}
}
