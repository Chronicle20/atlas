package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// comboDrainBuffWithAmount builds a live buff carrying a single COMBO_DRAIN
// stat change of the given amount. buff.NewBuff takes seven arguments — the
// trailing bool is noExpiry.
func comboDrainBuffWithAmount(amount int32) buff.Model {
	return buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeComboDrain), amount)},
		time.Now(), time.Now().Add(time.Hour), false)
}

// expiredComboDrainBuffWithAmount builds an already-expired COMBO_DRAIN buff.
func expiredComboDrainBuffWithAmount(amount int32) buff.Model {
	past := time.Now().Add(-time.Hour)
	return buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeComboDrain), amount)},
		past, past, false)
}

// attackWithDamages builds an AttackInfo of the given type with one
// DamageInfo per damages slice (monster ids 100, 101, ...).
func attackWithDamages(at packetmodel.AttackType, monsterDamages ...[]uint32) packetmodel.AttackInfo {
	ai := packetmodel.NewAttackInfo(at)
	for i, damages := range monsterDamages {
		di := packetmodel.NewDamageInfo(byte(len(damages))).
			SetMonsterId(uint32(100 + i)).
			SetDamages(damages)
		ai = ai.AddDamageInfo(*di)
	}
	return *ai
}

func TestBuffStatAmount(t *testing.T) {
	otherStat := buff.NewBuff(0, 1, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeSoulArrow), 1)},
		time.Now(), time.Now().Add(time.Hour), false)
	mixedStats := buff.NewBuff(0, 1, 0,
		[]stat.Model{
			stat.NewStat(string(charconst.TemporaryStatTypeSoulArrow), 1),
			stat.NewStat(string(charconst.TemporaryStatTypeComboDrain), 4),
		},
		time.Now(), time.Now().Add(time.Hour), false)

	tests := []struct {
		name       string
		buffs      []buff.Model
		wantAmount int32
		wantOk     bool
	}{
		{"present", []buff.Model{comboDrainBuffWithAmount(5)}, 5, true},
		{"absent - other stat only", []buff.Model{otherStat}, 0, false},
		{"nil slice", nil, 0, false},
		{"expired buff skipped", []buff.Model{expiredComboDrainBuffWithAmount(5)}, 0, false},
		{"expired then live - live wins", []buff.Model{expiredComboDrainBuffWithAmount(9), comboDrainBuffWithAmount(3)}, 3, true},
		{"first live match wins", []buff.Model{comboDrainBuffWithAmount(2), comboDrainBuffWithAmount(7)}, 2, true},
		{"matching stat alongside other stats", []buff.Model{mixedStats}, 4, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			amount, ok := buffStatAmount(tc.buffs, charconst.TemporaryStatTypeComboDrain)
			if ok != tc.wantOk || amount != tc.wantAmount {
				t.Fatalf("buffStatAmount = (%d, %v), want (%d, %v)", amount, ok, tc.wantAmount, tc.wantOk)
			}
		})
	}
}

func TestAttackTotalDamage(t *testing.T) {
	tests := []struct {
		name string
		ai   packetmodel.AttackInfo
		want uint64
	}{
		{"single monster single line", attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}), 1000},
		{"multi monster multi line", attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000, 2000}, []uint32{3000}), 6000},
		{"no damage entries", attackWithDamages(packetmodel.AttackTypeMelee), 0},
		{"large lines sum in uint64", attackWithDamages(packetmodel.AttackTypeMelee, []uint32{math.MaxUint32, math.MaxUint32}), 2 * uint64(math.MaxUint32)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := attackTotalDamage(tc.ai); got != tc.want {
				t.Fatalf("attackTotalDamage = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestComboDrainHealAmount(t *testing.T) {
	tests := []struct {
		name        string
		totalDamage uint64
		percent     int32
		want        int16
	}{
		{"nominal", 1000, 5, 50},
		{"integer truncation", 999, 3, 29},
		{"zero damage", 0, 5, 0},
		{"zero percent", 1000, 0, 0},
		{"negative percent", 1000, -5, 0},
		{"sub-1 result truncates to zero", 99, 1, 0},
		{"exact MaxInt16 unclamped", 3276700, 1, math.MaxInt16},
		{"one over boundary clamps", 3276800, 1, math.MaxInt16},
		{"huge total saturates without overflow", 15 * 15 * uint64(math.MaxUint32), 2147483647, math.MaxInt16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := comboDrainHealAmount(tc.totalDamage, tc.percent); got != tc.want {
				t.Fatalf("comboDrainHealAmount(%d, %d) = %d, want %d", tc.totalDamage, tc.percent, got, tc.want)
			}
		})
	}
}

// recordingChangeHP captures every emission comboDrainTryProc makes and can
// simulate a downstream failure.
type recordingChangeHP struct {
	calls []int16
	err   error
}

func (r *recordingChangeHP) fn(_ field.Model, _ uint32, amount int16) error {
	r.calls = append(r.calls, amount)
	return r.err
}

// countingBuffs serves a fixed buff slice (or error) and counts invocations.
type countingBuffs struct {
	buffs []buff.Model
	err   error
	calls int
}

func (c *countingBuffs) fn(_ uint32) ([]buff.Model, error) {
	c.calls++
	return c.buffs, c.err
}

func TestComboDrainTryProc(t *testing.T) {
	l := logrus.New()
	f := testField(100000000)

	tests := []struct {
		name      string
		buffs     []buff.Model
		buffErr   error
		ai        packetmodel.AttackInfo
		changeErr error
		wantCalls []int16
	}{
		{
			name:      "buff present single monster",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: []int16{50},
		},
		{
			// Pins the anti-Cosmic-quirk AC: one heal from the plain total
			// (6000 * 10 / 100 = 600), never per-monster running totals.
			name:      "buff present multi monster multi line - one call on plain total",
			buffs:     []buff.Model{comboDrainBuffWithAmount(10)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000, 2000}, []uint32{3000}),
			wantCalls: []int16{600},
		},
		{
			name:      "buff absent",
			buffs:     []buff.Model{},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: nil,
		},
		{
			name:      "buff fetch error",
			buffErr:   errors.New("buffs down"),
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: nil,
		},
		{
			name:      "expired buff only",
			buffs:     []buff.Model{expiredComboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			wantCalls: nil,
		},
		{
			name:      "zero total damage",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{0}),
			wantCalls: nil,
		},
		{
			name:      "heal truncates to zero",
			buffs:     []buff.Model{comboDrainBuffWithAmount(1)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{99}),
			wantCalls: nil,
		},
		{
			name:      "changeHP error swallowed - no panic no retry",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMelee, []uint32{1000}),
			changeErr: errors.New("kafka down"),
			wantCalls: []int16{50},
		},
		// Attack-type blindness (melee/ranged/magic/energy AC): the proc has
		// no type filter by construction; these pin that none creeps in.
		// Energy is the touch handler's type (character_attack_touch.go).
		{
			name:      "ranged attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeRanged, []uint32{1000}),
			wantCalls: []int16{50},
		},
		{
			name:      "magic attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeMagic, []uint32{1000}),
			wantCalls: []int16{50},
		},
		{
			name:      "energy (touch) attack heals",
			buffs:     []buff.Model{comboDrainBuffWithAmount(5)},
			ai:        attackWithDamages(packetmodel.AttackTypeEnergy, []uint32{1000}),
			wantCalls: []int16{50},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingChangeHP{err: tc.changeErr}
			cb := &countingBuffs{buffs: tc.buffs, err: tc.buffErr}
			comboDrainTryProc(l, cb.fn, rec.fn, f, 42, tc.ai)
			if cb.calls > 1 {
				t.Fatalf("getBuffs called %d times, want at most 1", cb.calls)
			}
			if len(rec.calls) != len(tc.wantCalls) {
				t.Fatalf("changeHP called %d times (%v), want %d (%v)", len(rec.calls), rec.calls, len(tc.wantCalls), tc.wantCalls)
			}
			for i := range tc.wantCalls {
				if rec.calls[i] != tc.wantCalls[i] {
					t.Fatalf("changeHP call %d amount = %d, want %d", i, rec.calls[i], tc.wantCalls[i])
				}
			}
		})
	}
}
