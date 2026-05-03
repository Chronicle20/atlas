package handler

import (
	"atlas-channel/asset"
	"atlas-channel/compartment"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	channelinv "atlas-channel/inventory"
	"atlas-channel/monster"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
		EffectId:       uuid.NewString(),
		Statuses:       map[string]int32{"WEAPON_REFLECT": 1},
		Duration:       60000,
		ReflectKind:    monster2.ReflectKindPhysical,
		ReflectPercent: 30,
		ReflectLtX:     -100,
		ReflectLtY:     -100,
		ReflectRbX:     100,
		ReflectRbY:     100,
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
	r, within := computeReflect([]int32{1000}, info, /*charX*/ 50, 0, /*monX*/ 0, 0)
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

// applyStatusCall captures one invocation of the applyStatus closure so tests
// can assert on (monsterId, statuses, duration) without inspecting Kafka.
type applyStatusCall struct {
	monsterId   uint32
	characterId uint32
	skillId     uint32
	skillLevel  uint32
	statuses    map[string]int32
	duration    uint32
}

// damageEntryFakes is a minimal in-memory recorder used by Doom helper tests.
// Only Doom-relevant interactions are tracked.
type damageEntryFakes struct {
	applyStatusCalls       []applyStatusCall
	applyDamageCalls       int
	emitReflectDamageCalls int
	reflects               map[uint32]monster.ReflectInfo
	monsters               map[uint32]monster.Model
}

func (df *damageEntryFakes) deps() damageInfoEntryDeps {
	return damageInfoEntryDeps{
		getReflect: func(_ tenant.Model, monsterId uint32, _ string) (monster.ReflectInfo, bool) {
			ri, ok := df.reflects[monsterId]
			return ri, ok
		},
		getMonster: func(monsterId uint32) (monster.Model, error) {
			m, ok := df.monsters[monsterId]
			if !ok {
				return monster.Model{}, errors.New("not found")
			}
			return m, nil
		},
		applyDamage: func(_ field.Model, _ uint32, _ uint32, _ []uint32, _ byte) error {
			df.applyDamageCalls++
			return nil
		},
		emitReflectDamage: func(_ field.Model, _ uint32, _ uint32, _ uint32, _ uint32, _ string) error {
			df.emitReflectDamageCalls++
			return nil
		},
		applyStatus: func(_ field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error {
			df.applyStatusCalls = append(df.applyStatusCalls, applyStatusCall{
				monsterId:   monsterId,
				characterId: characterId,
				skillId:     skillId,
				skillLevel:  skillLevel,
				statuses:    statuses,
				duration:    duration,
			})
			return nil
		},
		loadVenomStats: func() effective_stats.RestModel { return effective_stats.RestModel{} },
	}
}

func newDoomEffect() effect.Model {
	se, _ := effect.Extract(effect.RestModel{
		Duration:      20000,
		MonsterStatus: map[string]uint32{monster2.StatusDoom: 1},
	})
	return se
}

func newDoomAttackInfo(monsterIds ...uint32) packetmodel.AttackInfo {
	aip := packetmodel.NewAttackInfo(packetmodel.AttackTypeMagic).SetSkillId(uint32(skillconst.PriestDoomId))
	for _, mid := range monsterIds {
		dip := packetmodel.NewDamageInfo(0).SetMonsterId(mid).SetDamages(nil)
		aip.AddDamageInfo(*dip)
	}
	return *aip
}

// TestProcessDamageInfoEntry_Doom_EmptyDamagesAppliesStatus verifies the
// happy path: an empty-damage DOOM-bearing attack lands on a target with no
// reflect, producing exactly one applyStatus call with the DOOM map and the
// effect's duration.
func TestProcessDamageInfoEntry_Doom_EmptyDamagesAppliesStatus(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l := logrus.New()
	l.Out = io.Discard

	ai := newDoomAttackInfo(1)
	di := ai.DamageInfo()[0]
	se := newDoomEffect()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	df := &damageEntryFakes{}
	processDamageInfoEntry(l, di, ai, se, 30, /*casterId*/ 1001, 0, 0, f, tm, "MAGICAL", df.deps())

	if len(df.applyStatusCalls) != 1 {
		t.Fatalf("applyStatus calls = %d, want 1 (%v)", len(df.applyStatusCalls), df.applyStatusCalls)
	}
	got := df.applyStatusCalls[0]
	if got.monsterId != 1 || got.skillId != uint32(skillconst.PriestDoomId) || got.skillLevel != 30 {
		t.Errorf("applyStatus args = %+v, want monsterId=1 skillId=%d skillLevel=30", got, uint32(skillconst.PriestDoomId))
	}
	if got.statuses[monster2.StatusDoom] != 1 {
		t.Errorf("statuses[DOOM] = %d, want 1", got.statuses[monster2.StatusDoom])
	}
	if got.duration != 20000 {
		t.Errorf("duration = %d, want 20000", got.duration)
	}
	if df.applyDamageCalls != 0 {
		t.Errorf("applyDamage called %d times, want 0", df.applyDamageCalls)
	}
	if df.emitReflectDamageCalls != 0 {
		t.Errorf("emitReflectDamage called %d times, want 0", df.emitReflectDamageCalls)
	}
}

// TestProcessDamageInfoEntry_Doom_BlockedByReflect verifies the new
// Doom-gated probe: when the target has a magic-reflect window and the
// inbound status set is DOOM-bearing, the apply is skipped (no reflect
// damage is emitted because Doom does no damage).
func TestProcessDamageInfoEntry_Doom_BlockedByReflect(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l := logrus.New()
	l.Out = io.Discard

	ai := newDoomAttackInfo(1)
	di := ai.DamageInfo()[0]
	se := newDoomEffect()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	df := &damageEntryFakes{
		reflects: map[uint32]monster.ReflectInfo{
			1: {Kind: monster2.ReflectKindMagical, Percent: 30, LtX: -100, LtY: -100, RbX: 100, RbY: 100, MaxDamage: 9999, ExpiresAt: time.Now().Add(time.Minute)},
		},
	}
	processDamageInfoEntry(l, di, ai, se, 30, 1001, 0, 0, f, tm, monster2.ReflectKindMagical, df.deps())

	if len(df.applyStatusCalls) != 0 {
		t.Errorf("applyStatus calls = %d, want 0 (Doom blocked by reflect)", len(df.applyStatusCalls))
	}
	if df.emitReflectDamageCalls != 0 {
		t.Errorf("emitReflectDamage calls = %d, want 0 (Doom does no damage to reflect)", df.emitReflectDamageCalls)
	}
	if df.applyDamageCalls != 0 {
		t.Errorf("applyDamage calls = %d, want 0 (no damage path)", df.applyDamageCalls)
	}
}

// TestProcessDamageInfoEntry_Doom_MultiTargetSpread verifies the spread case:
// three Doom targets, the middle one carries a magic-reflect window, the
// other two are clean. Helper invoked once per DamageInfo. Result: exactly
// two applyStatus calls (monsters 1 and 3); none for monster 2.
func TestProcessDamageInfoEntry_Doom_MultiTargetSpread(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l := logrus.New()
	l.Out = io.Discard

	ai := newDoomAttackInfo(1, 2, 3)
	se := newDoomEffect()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	df := &damageEntryFakes{
		reflects: map[uint32]monster.ReflectInfo{
			2: {Kind: monster2.ReflectKindMagical, Percent: 30, LtX: -100, LtY: -100, RbX: 100, RbY: 100, MaxDamage: 9999, ExpiresAt: time.Now().Add(time.Minute)},
		},
	}
	for _, di := range ai.DamageInfo() {
		processDamageInfoEntry(l, di, ai, se, 30, 1001, 0, 0, f, tm, monster2.ReflectKindMagical, df.deps())
	}

	if len(df.applyStatusCalls) != 2 {
		t.Fatalf("applyStatus calls = %d, want 2 (%v)", len(df.applyStatusCalls), df.applyStatusCalls)
	}
	gotIds := []uint32{df.applyStatusCalls[0].monsterId, df.applyStatusCalls[1].monsterId}
	if !(gotIds[0] == 1 && gotIds[1] == 3) {
		t.Errorf("applyStatus monster ids = %v, want [1 3] (monster 2 reflect-blocked)", gotIds)
	}
	if df.emitReflectDamageCalls != 0 {
		t.Errorf("emitReflectDamage calls = %d, want 0", df.emitReflectDamageCalls)
	}
	if df.applyDamageCalls != 0 {
		t.Errorf("applyDamage calls = %d, want 0", df.applyDamageCalls)
	}
}

// TestProcessDamageInfoEntry_NonDoom_EmptyDamagesIgnoresReflectProbe pins
// that the new probe is Doom-gated. A hypothetical empty-damage status that
// is not DOOM should still apply through a magic-reflect window.
func TestProcessDamageInfoEntry_NonDoom_EmptyDamagesIgnoresReflectProbe(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	l := logrus.New()
	l.Out = io.Discard

	se, _ := effect.Extract(effect.RestModel{
		Duration:      5000,
		MonsterStatus: map[string]uint32{"FREEZE": 1},
	})
	aip := packetmodel.NewAttackInfo(packetmodel.AttackTypeMagic).SetSkillId(0)
	dip := packetmodel.NewDamageInfo(0).SetMonsterId(7).SetDamages(nil)
	aip.AddDamageInfo(*dip)
	ai := *aip
	di := ai.DamageInfo()[0]
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	df := &damageEntryFakes{
		reflects: map[uint32]monster.ReflectInfo{
			7: {Kind: monster2.ReflectKindMagical, Percent: 30, LtX: -100, LtY: -100, RbX: 100, RbY: 100, MaxDamage: 9999, ExpiresAt: time.Now().Add(time.Minute)},
		},
	}
	processDamageInfoEntry(l, di, ai, se, 1, 1001, 0, 0, f, tm, monster2.ReflectKindMagical, df.deps())

	if len(df.applyStatusCalls) != 1 {
		t.Errorf("non-Doom empty-damage status should apply through reflect; applyStatus calls = %d, want 1", len(df.applyStatusCalls))
	}
}

// TestFindItemSlotInInventory_Found verifies the helper returns the slot of
// the first asset in the resolved compartment whose template id equals the
// queried item id.
func TestFindItemSlotInInventory_Found(t *testing.T) {
	const magicRockId = uint32(4006000)
	compId := uuid.New()
	a := asset.NewBuilder(compId, magicRockId).
		SetId(101).
		SetSlot(3).
		SetQuantity(5).
		MustBuild()
	useComp := compartment.NewBuilder(compId, 1, inventoryconst.TypeValueETC, 24).
		SetAssets([]asset.Model{a}).
		MustBuild()
	inv := channelinv.NewBuilder(1).
		SetCompartment(useComp).
		MustBuild()

	pos, found := findItemSlotInInventory(inv, magicRockId)
	if !found {
		t.Fatalf("expected to find item [%d]", magicRockId)
	}
	if pos != slot.Position(3) {
		t.Errorf("pos = %d, want 3", pos)
	}
}

// TestFindItemSlotInInventory_NotFound pins the absent-item branch: if the
// inventory has no matching template id in the resolved compartment, the
// helper returns (0, false). The caller logs a warning and the cast is still
// permitted (defense-in-depth, not authoritative).
func TestFindItemSlotInInventory_NotFound(t *testing.T) {
	const magicRockId = uint32(4006000)
	compId := uuid.New()
	useComp := compartment.NewBuilder(compId, 1, inventoryconst.TypeValueETC, 24).
		MustBuild()
	inv := channelinv.NewBuilder(1).
		SetCompartment(useComp).
		MustBuild()

	if _, found := findItemSlotInInventory(inv, magicRockId); found {
		t.Errorf("expected not-found for empty ETC compartment")
	}
}
