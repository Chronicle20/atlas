package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestIsDrainSkill(t *testing.T) {
	tests := []struct {
		name string
		id   skill3.Id
		want bool
	}{
		{"assassin drain", skill3.AssassinDrainId, true},
		{"marauder energy drain", skill3.MarauderEnergyDrainId, true},
		{"thunder breaker energy drain", skill3.ThunderBreakerStage3EnergyDrainId, true},
		{"night walker vampire", skill3.NightWalkerStage2VampireId, true},
		{"aran combo drain is NOT attack-side drain", skill3.AranStage2ComboDrainId, false},
		{"zero id", skill3.Id(0), false},
		{"adjacent id", skill3.AssassinDrainId + 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDrainSkill(tc.id); got != tc.want {
				t.Errorf("isDrainSkill(%d) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestDrainHealAmount(t *testing.T) {
	const big = uint32(2_000_000_000) // caps well above every raw heal below

	tests := []struct {
		name           string
		totalDamage    uint32
		x              int16
		monsterMaxHp   uint32
		effectiveMaxHp uint32
		want           int16
	}{
		// FR-3 spot values: percentage math with no cap engaged.
		{"drain L30 x=45: 1000 dmg heals 450", 1000, 45, big, big, 450},
		{"drain L1 x=16 floor: 333*16/100 = 53", 333, 16, big, big, 53},
		{"vampire L20 x=10: 5000 dmg heals 500", 5000, 10, big, big, 500},
		{"energy drain L20 x=20: 12345*20/100 = 2469", 12345, 20, big, big, 2469},
		// Caps.
		{"monster max HP caps the heal", 10000, 45, 100, big, 100},
		{"half effective max HP caps the heal", 10000, 45, big, 2000, 1000},
		{"half-cap floors odd effectiveMaxHp: 2001/2 = 1000", 10000, 45, big, 2001, 1000},
		{"tighter of the two caps wins", 10000, 45, 300, 2000, 300},
		// Zero guards.
		{"zero damage heals nothing", 0, 45, big, big, 0},
		{"x=0 heals nothing", 1000, 0, big, big, 0},
		{"negative x heals nothing", 1000, -5, big, big, 0},
		{"effectiveMaxHp=0 (stats fetch failed) heals nothing", 1000, 45, big, 0, 0},
		{"monsterMaxHp=0 heals nothing", 1000, 45, 0, big, 0},
		// Defensive int16 clamp with pathological inputs.
		{"int16 clamp on pathological damage", 4_000_000_000, 100, 4_294_967_295, 4_294_967_295, 32767},
		{"raw heal exactly at MaxInt16 passes through", 32767, 100, big, big, 32767},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := drainHealAmount(tc.totalDamage, tc.x, tc.monsterMaxHp, tc.effectiveMaxHp)
			if got != tc.want {
				t.Errorf("drainHealAmount(%d, %d, %d, %d) = %d, want %d",
					tc.totalDamage, tc.x, tc.monsterMaxHp, tc.effectiveMaxHp, got, tc.want)
			}
		})
	}
}

func discardLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testDrainField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

// TestOnDamageApplied_ReceivesSummedDamageTotal pins the widened hook
// contract: one invocation per damage-carrying entry, carrying the sum
// of that entry's damage lines.
func TestOnDamageApplied_ReceivesSummedDamageTotal(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(2).SetMonsterId(4001).SetDamages([]uint32{100, 250})

	var gotMonsterId, gotTotal uint32
	calls := 0
	deps := damageInfoEntryDeps{
		applyDamage: func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error { return nil },
		onDamageApplied: func(monsterId uint32, totalDamage uint32) {
			calls++
			gotMonsterId = monsterId
			gotTotal = totalDamage
		},
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, testDrainField(), testTenant(t), "", deps)

	if calls != 1 {
		t.Fatalf("onDamageApplied calls = %d, want 1", calls)
	}
	if gotMonsterId != 4001 {
		t.Errorf("monsterId = %d, want 4001", gotMonsterId)
	}
	if gotTotal != 350 {
		t.Errorf("totalDamage = %d, want 350", gotTotal)
	}
}

// TestOnDamageApplied_NotCalledForZeroDamageEntry: a status-only entry
// (no damage lines) never reaches the hook.
func TestOnDamageApplied_NotCalledForZeroDamageEntry(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(0).SetMonsterId(4002).SetDamages(nil)

	called := false
	deps := damageInfoEntryDeps{
		applyDamage:     func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error { return nil },
		onDamageApplied: func(_ uint32, _ uint32) { called = true },
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, testDrainField(), testTenant(t), "", deps)

	if called {
		t.Fatalf("onDamageApplied fired for a zero-damage entry")
	}
}

// TestOnDamageApplied_NotCalledForReflectedEntry: a reflected entry deals
// no damage, so the hook must not fire (drain inherits this for free).
func TestOnDamageApplied_NotCalledForReflectedEntry(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(1).SetMonsterId(4003).SetDamages([]uint32{500})
	f := testDrainField()

	called := false
	damaged := false
	deps := damageInfoEntryDeps{
		getReflect: func(_ tenant.Model, _ uint32, _ string) (monster.ReflectInfo, bool) {
			return monster.ReflectInfo{
				Kind:      monster2.ReflectKindPhysical,
				Percent:   30,
				LtX:       -100,
				LtY:       -100,
				RbX:       100,
				RbY:       100,
				MaxDamage: 9999,
			}, true
		},
		getMonster: func(monsterId uint32) (monster.Model, error) {
			return monster.NewModelBuilder(monsterId, f, 100100).Build()
		},
		applyDamage: func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error {
			damaged = true
			return nil
		},
		emitReflectDamage: func(_ field.Model, _, _, _ uint32, _ uint32, _ string) error { return nil },
		onDamageApplied:   func(_ uint32, _ uint32) { called = true },
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, f, testTenant(t), monster2.ReflectKindPhysical, deps)

	if damaged {
		t.Fatalf("applyDamage fired for a reflected entry")
	}
	if called {
		t.Fatalf("onDamageApplied fired for a reflected entry")
	}
}
