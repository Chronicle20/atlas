package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"math"
	"testing"
	"time"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
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
