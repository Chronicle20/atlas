package handler

import (
	"errors"
	"math"
	"testing"
	"time"

	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	"atlas-channel/monster"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2const "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestMortalBlowEligible(t *testing.T) {
	cases := []struct {
		name  string
		hp    uint32
		maxHp uint32
		x     int16
		want  bool
	}{
		// maxHp=999, x=20 -> threshold 999*20/100 = 199 (truncating division,
		// Cosmic parity: (getStats().getHp() * getX()) / 100).
		{"hp exactly at threshold", 199, 999, 20, true},
		{"hp one above threshold", 200, 999, 20, false},
		{"hp well below threshold", 1, 999, 20, true},
		{"x zero never eligible", 1, 999, 0, false},
		{"x negative never eligible", 1, 999, -5, false},
		{"maxHp zero never eligible", 0, 0, 20, false},
		// uint64 widening: maxHp near MaxUint32 must not overflow.
		// threshold = MaxUint32*50/100 = 2147483647 (floor).
		{"no overflow at MaxUint32", 2147483647, math.MaxUint32, 50, true},
		{"no overflow above threshold", 2147483648, math.MaxUint32, 50, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mortalBlowEligible(tc.hp, tc.maxHp, tc.x); got != tc.want {
				t.Fatalf("mortalBlowEligible(%d, %d, %d) = %v; want %v", tc.hp, tc.maxHp, tc.x, got, tc.want)
			}
		})
	}
}

func TestMortalBlowKillRoll(t *testing.T) {
	cases := []struct {
		name string
		roll int
		y    int16
		want bool
	}{
		{"roll equal to y procs", 5, 5, true},
		{"roll one above y misses", 6, 5, false},
		{"roll below y procs", 1, 5, true},
		{"y zero never procs", 1, 0, false},
		{"y negative never procs", 1, -1, false},
		{"y 100 always procs", 100, 100, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mortalBlowKillRoll(tc.roll, tc.y); got != tc.want {
				t.Fatalf("mortalBlowKillRoll(%d, %d) = %v; want %v", tc.roll, tc.y, got, tc.want)
			}
		})
	}
}

func TestIsMortalBlowAttack(t *testing.T) {
	cases := []struct {
		name    string
		at      packetmodel.AttackType
		skillId uint32
		want    bool
	}{
		{"ranged + Ranger Mortal Blow", packetmodel.AttackTypeRanged, uint32(skill3.RangerMortalBlowId), true},
		{"ranged + Sniper Mortal Blow", packetmodel.AttackTypeRanged, uint32(skill3.SniperMortalBlowId), true},
		{"ranged + other skill", packetmodel.AttackTypeRanged, uint32(skill3.RangerStrafeId), false},
		{"ranged + no skill", packetmodel.AttackTypeRanged, 0, false},
		{"melee + Mortal Blow id", packetmodel.AttackTypeMelee, uint32(skill3.RangerMortalBlowId), false},
		{"magic + Mortal Blow id", packetmodel.AttackTypeMagic, uint32(skill3.RangerMortalBlowId), false},
		{"energy + Mortal Blow id", packetmodel.AttackTypeEnergy, uint32(skill3.SniperMortalBlowId), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMortalBlowAttack(tc.at, tc.skillId); got != tc.want {
				t.Fatalf("isMortalBlowAttack(%v, %d) = %v; want %v", tc.at, tc.skillId, got, tc.want)
			}
		})
	}
}

// --- mortalBlowTryProc flow tests -------------------------------------
//
// The deps struct fakes the monster snapshot, the KILL emit, and the RNG,
// so every branch is pinned deterministically (FR-8).

func mbEffect(t *testing.T, x int16, y int16) effect.Model {
	t.Helper()
	m, err := effect.Extract(effect.RestModel{X: x, Y: y})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return m
}

func mbField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func mbMonster(t *testing.T, uniqueId uint32, hp uint32, maxHp uint32) monster.Model {
	t.Helper()
	return monster.NewModelBuilder(uniqueId, mbField(), 1000000).SetHp(hp).SetMaxHp(maxHp).MustBuild()
}

// TestMortalBlowTryProc_InertEffectSkipsSnapshotFetch — x/y ≤ 0 in tenant
// data means the passive is inert: no snapshot fetch, no roll, no emit.
func TestMortalBlowTryProc_InertEffectSkipsSnapshotFetch(t *testing.T) {
	for _, se := range []effect.Model{mbEffect(t, 0, 5), mbEffect(t, 20, 0)} {
		fetched := false
		deps := mortalBlowDeps{
			getMonster: func(uint32) (monster.Model, error) {
				fetched = true
				return monster.Model{}, nil
			},
			emitKill: func(field.Model, uint32, uint32) error {
				t.Fatal("emitKill must not be called for inert effect")
				return nil
			},
			roll: func() int { t.Fatal("roll must not be called for inert effect"); return 0 },
		}
		mortalBlowTryProc(logrus.New(), deps, se, 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
		if fetched {
			t.Fatal("snapshot must not be fetched for inert effect")
		}
	}
}

// TestMortalBlowTryProc_SnapshotErrorSwallowed — a failed monster snapshot
// fetch (despawn race) is logged and swallowed; no roll, no emit (FR-5).
func TestMortalBlowTryProc_SnapshotErrorSwallowed(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) {
			return monster.Model{}, errors.New("monster gone")
		},
		emitKill: func(field.Model, uint32, uint32) error {
			t.Fatal("emitKill must not be called on snapshot error")
			return nil
		},
		roll: func() int { t.Fatal("roll must not be called on snapshot error"); return 0 },
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// TestMortalBlowTryProc_AboveThresholdNoRoll — monster HP above
// maxHp·x/100 never rolls (maxHp=1000, x=20 -> threshold 200; hp=201).
func TestMortalBlowTryProc_AboveThresholdNoRoll(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 201, 1000), nil },
		emitKill: func(field.Model, uint32, uint32) error {
			t.Fatal("emitKill must not be called above threshold")
			return nil
		},
		roll: func() int { t.Fatal("roll must not be called above threshold"); return 0 },
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// TestMortalBlowTryProc_RollFailNoEmit — at threshold, roll y+1 misses.
func TestMortalBlowTryProc_RollFailNoEmit(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 200, 1000), nil },
		emitKill: func(field.Model, uint32, uint32) error {
			t.Fatal("emitKill must not be called on failed roll")
			return nil
		},
		roll: func() int { return 6 }, // y=5 -> 6 misses
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// TestMortalBlowTryProc_ProcEmitsKill — at threshold with roll == y the
// kill is emitted with the caster and monster. The skill id is not sent
// over the KILL wire (traceability-only; see KillCommandBody), so the
// emit seam no longer carries it.
func TestMortalBlowTryProc_ProcEmitsKill(t *testing.T) {
	var gotMonster, gotCharacter uint32
	emitted := false
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 200, 1000), nil },
		emitKill: func(_ field.Model, monsterId uint32, characterId uint32) error {
			emitted = true
			gotMonster, gotCharacter = monsterId, characterId
			return nil
		},
		roll: func() int { return 5 }, // y=5 -> 5 procs
	}
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.SniperMortalBlowId))
	if !emitted {
		t.Fatal("expected KILL emit")
	}
	if gotMonster != 42 || gotCharacter != 7 {
		t.Fatalf("emitKill(monster=%d, character=%d), want (42, 7)", gotMonster, gotCharacter)
	}
}

// TestMortalBlowTryProc_EmitErrorSwallowed — a failed KILL emit is logged
// and swallowed; mortalBlowTryProc returns normally (FR-5).
func TestMortalBlowTryProc_EmitErrorSwallowed(t *testing.T) {
	deps := mortalBlowDeps{
		getMonster: func(uint32) (monster.Model, error) { return mbMonster(t, 42, 200, 1000), nil },
		emitKill: func(field.Model, uint32, uint32) error {
			return errors.New("kafka down")
		},
		roll: func() int { return 1 },
	}
	// Must not panic and must return normally.
	mortalBlowTryProc(logrus.New(), deps, mbEffect(t, 20, 5), 42, mbField(), 7, uint32(skill3.RangerMortalBlowId))
}

// --- onDamageApplied gating through processDamageInfoEntry -------------
//
// Reflected and status-only entries must never reach onDamageApplied (and
// therefore never proc Mortal Blow); a plain damage entry must reach it.

func mbEntryDeps(onDamageApplied func(di packetmodel.DamageInfo, totalDamage uint32)) damageInfoEntryDeps {
	return damageInfoEntryDeps{
		getReflect: func(tenant.Model, uint32, string) (monster.ReflectInfo, bool) {
			return monster.ReflectInfo{}, false
		},
		getMonster: func(uint32) (monster.LiveEntry, error) { return monster.LiveEntry{}, nil },
		applyDamage: func(field.Model, uint32, uint32, []uint32, byte) error {
			return nil
		},
		emitReflectDamage: func(field.Model, uint32, uint32, uint32, uint32, string) error {
			return nil
		},
		applyStatus: func(field.Model, uint32, uint32, uint32, uint32, map[string]int32, uint32) error {
			return nil
		},
		loadEffectiveStats: func() effective_stats.RestModel { return effective_stats.RestModel{} },
		onDamageApplied:    onDamageApplied,
	}
}

// TestProcessDamageInfoEntry_DamageEntryReachesOnDamageApplied — the happy
// path invokes the callback once with the entry's monster id.
func TestProcessDamageInfoEntry_DamageEntryReachesOnDamageApplied(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	var got []uint32
	deps := mbEntryDeps(func(di packetmodel.DamageInfo, totalDamage uint32) { got = append(got, di.MonsterId()) })

	di := *packetmodel.NewDamageInfo(1).SetMonsterId(42).SetDamages([]uint32{100})
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged).SetSkillId(uint32(skill3.RangerMortalBlowId))
	processDamageInfoEntry(logrus.New(), di, ai, mbEffect(t, 20, 5), 1, 7, 0, 0, mbField(), tm, monster2const.ReflectKindPhysical, deps)

	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("onDamageApplied calls = %v, want [42]", got)
	}
}

// TestProcessDamageInfoEntry_StatusOnlyEntrySkipsOnDamageApplied — an entry
// with no damage lines never reaches the proc callback.
func TestProcessDamageInfoEntry_StatusOnlyEntrySkipsOnDamageApplied(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	invoked := false
	deps := mbEntryDeps(func(di packetmodel.DamageInfo, totalDamage uint32) { invoked = true })

	di := *packetmodel.NewDamageInfo(0).SetMonsterId(42)
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged).SetSkillId(uint32(skill3.RangerMortalBlowId))
	processDamageInfoEntry(logrus.New(), di, ai, mbEffect(t, 20, 5), 1, 7, 0, 0, mbField(), tm, monster2const.ReflectKindPhysical, deps)

	if invoked {
		t.Fatal("onDamageApplied must not run for a status-only entry")
	}
}

// TestProcessDamageInfoEntry_ReflectedEntrySkipsOnDamageApplied — when the
// monster reflects the hit, damage is not applied and the proc callback
// never runs.
func TestProcessDamageInfoEntry_ReflectedEntrySkipsOnDamageApplied(t *testing.T) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	invoked := false
	deps := mbEntryDeps(func(di packetmodel.DamageInfo, totalDamage uint32) { invoked = true })
	deps.getReflect = func(tenant.Model, uint32, string) (monster.ReflectInfo, bool) {
		return monster.ReflectInfo{
			Kind:      monster2const.ReflectKindPhysical,
			Percent:   30,
			LtX:       -100,
			LtY:       -100,
			RbX:       100,
			RbY:       100,
			MaxDamage: 9999,
			ExpiresAt: time.Now().Add(time.Minute),
		}, true
	}
	deps.getMonster = func(uint32) (monster.LiveEntry, error) {
		return monster.LiveEntryFromModel(mbMonster(t, 42, 500, 1000)), nil
	}

	di := *packetmodel.NewDamageInfo(1).SetMonsterId(42).SetDamages([]uint32{100})
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged).SetSkillId(uint32(skill3.RangerMortalBlowId))
	processDamageInfoEntry(logrus.New(), di, ai, mbEffect(t, 20, 5), 1, 7, 0, 0, mbField(), tm, monster2const.ReflectKindPhysical, deps)

	if invoked {
		t.Fatal("onDamageApplied must not run for a reflected entry")
	}
}
