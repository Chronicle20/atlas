package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"math"
	"testing"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

const (
	tamedMountSkillId     = uint32(skill2.BeginnerMonsterRidingId) // 1004
	skillOnlyMountSkillId = uint32(skill2.BeginnerBroomstickId)    // 1019 -> vehicle 1932005
	tamingMobItemId       = int32(1902000)                         // arbitrary equipped taming-mob id
)

// recordingDeps captures collaborator invocations so each of the five mount
// cases can be asserted offline without Kafka, REST, or a session.
type recordingDeps struct {
	mounted     bool
	mountedErr  error
	equip       map[int16]int32 // slot position -> taming-mob/saddle item id
	equipErr    map[int16]error
	applyCalled  bool
	applyAmount  int32
	applyStatups []statup.Model
	applySource  int32
	applyDur     int32
	cancelCount  int
	cancelSrc    int32
}

func (d *recordingDeps) mountDeps() mountDeps {
	return mountDeps{
		isMounted: func(characterId uint32, sourceId int32) (bool, error) {
			return d.mounted, d.mountedErr
		},
		equipInSlot: func(characterId uint32, pos int16) (int32, bool, error) {
			if d.equipErr != nil {
				if err, ok := d.equipErr[pos]; ok {
					return 0, false, err
				}
			}
			if id, ok := d.equip[pos]; ok {
				return id, true, nil
			}
			return 0, false, nil
		},
		applyBuff: func(f field.Model, characterId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) error {
			d.applyCalled = true
			d.applySource = sourceId
			d.applyDur = duration
			d.applyStatups = statups
			if len(statups) > 0 {
				d.applyAmount = statups[0].Amount()
			}
			return nil
		},
		cancelBuff: func(f field.Model, characterId uint32, sourceId int32) error {
			d.cancelCount++
			d.cancelSrc = sourceId
			return nil
		},
	}
}

func mountInfo(skillId uint32) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().SetSkillId(skillId).SetSkillLevel(1).Build()
}

func mountEffect(statups []statup.RestModel) effect.Model {
	e, err := effect.Extract(effect.RestModel{Statups: statups})
	if err != nil {
		panic(err)
	}
	return e
}

func vehicleStatup(amount int32) []statup.RestModel {
	return []statup.RestModel{{Type: string(charconst.TemporaryStatTypeMonsterRiding), Amount: amount}}
}

// findStatupAmount returns the amount of the statup of the given type, or
// (0, false) when absent.
func findStatupAmount(sus []statup.Model, typ string) (int32, bool) {
	for _, su := range sus {
		if su.Mask() == typ {
			return su.Amount(), true
		}
	}
	return 0, false
}

func TestMountToggleCancelsWhenAlreadyMounted(t *testing.T) {
	d := &recordingDeps{
		mounted: true,
		equip:   map[int16]int32{-18: tamingMobItemId, -19: 1902020},
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if d.applyCalled {
		t.Errorf("expected Apply NOT called when already mounted")
	}
	if d.cancelCount != 1 {
		t.Errorf("expected Cancel called once, got %d", d.cancelCount)
	}
	if d.cancelSrc != int32(tamedMountSkillId) {
		t.Errorf("Cancel sourceId = %d, want %d", d.cancelSrc, tamedMountSkillId)
	}
}

func TestMountTamedRequiresBothSlots(t *testing.T) {
	d := &recordingDeps{
		mounted: false,
		equip:   map[int16]int32{-18: tamingMobItemId}, // -19 empty
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if d.applyCalled {
		t.Errorf("expected no Apply when saddle slot -19 is empty")
	}
	if d.cancelCount != 0 {
		t.Errorf("expected no Cancel, got %d", d.cancelCount)
	}
}

func TestMountTamedAppliesVehicleFromSlot18(t *testing.T) {
	d := &recordingDeps{
		mounted: false,
		equip:   map[int16]int32{-18: tamingMobItemId, -19: 1902020},
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if !d.applyCalled {
		t.Fatalf("expected Apply called with both slots present")
	}
	if d.applyAmount != tamingMobItemId {
		t.Errorf("Apply amount = %d, want taming-mob id %d", d.applyAmount, tamingMobItemId)
	}
	if d.applySource != int32(tamedMountSkillId) {
		t.Errorf("Apply sourceId = %d, want skillId %d", d.applySource, tamedMountSkillId)
	}
	if d.applyDur != int32(math.MaxInt32) {
		t.Errorf("Apply duration = %d, want MaxInt32 %d", d.applyDur, int32(math.MaxInt32))
	}
	if d.cancelCount != 0 {
		t.Errorf("expected no Cancel, got %d", d.cancelCount)
	}
}

func TestMountTamedSlot18EmptyNoOp(t *testing.T) {
	d := &recordingDeps{
		mounted: false,
		equip:   map[int16]int32{-19: 1902020}, // -18 empty
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if d.applyCalled {
		t.Errorf("expected no Apply when taming-mob slot -18 is empty")
	}
	if d.cancelCount != 0 {
		t.Errorf("expected no Cancel, got %d", d.cancelCount)
	}
}

func TestMountSkillOnlyNoSlotCheck(t *testing.T) {
	const vehicleId = int32(1932005) // Broomstick vehicle id from skill effect data
	d := &recordingDeps{
		mounted: false,
		// No equip entries at all: skill-only mounts must not read slots.
		equipErr: map[int16]error{-18: errStub, -19: errStub},
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(skillOnlyMountSkillId), mountEffect(vehicleStatup(vehicleId)), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if !d.applyCalled {
		t.Fatalf("expected Apply called for skill-only mount")
	}
	if d.applyAmount != vehicleId {
		t.Errorf("Apply amount = %d, want MONSTER_RIDING amount from StatUps %d", d.applyAmount, vehicleId)
	}
	if d.applySource != int32(skillOnlyMountSkillId) {
		t.Errorf("Apply sourceId = %d, want skillId %d", d.applySource, skillOnlyMountSkillId)
	}
	if d.applyDur != int32(math.MaxInt32) {
		t.Errorf("Apply duration = %d, want MaxInt32 %d", d.applyDur, int32(math.MaxInt32))
	}
}

// TestMountSkillOnlyAppliesAllStatups verifies that a skill-only mount applies
// the full effect statup set — the vehicle id AND the skill's other granted
// stats (e.g. +10 weapon/magic defense for the Yeti Rider). Regression for
// mounts dropping every statup except MONSTER_RIDING.
func TestMountSkillOnlyAppliesAllStatups(t *testing.T) {
	const vehicleId = int32(1932005)
	statups := []statup.RestModel{
		{Type: string(charconst.TemporaryStatTypeWeaponDefense), Amount: 10},
		{Type: string(charconst.TemporaryStatTypeMagicDefense), Amount: 10},
		{Type: string(charconst.TemporaryStatTypeMonsterRiding), Amount: vehicleId},
	}
	d := &recordingDeps{
		mounted:  false,
		equipErr: map[int16]error{-18: errStub, -19: errStub},
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(skillOnlyMountSkillId), mountEffect(statups), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if !d.applyCalled {
		t.Fatalf("expected Apply called for skill-only mount")
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeMonsterRiding)); !ok || amt != vehicleId {
		t.Errorf("MONSTER_RIDING amount = %d (present=%v), want vehicle id %d", amt, ok, vehicleId)
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeWeaponDefense)); !ok || amt != 10 {
		t.Errorf("WEAPON_DEFENSE amount = %d (present=%v), want 10", amt, ok)
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeMagicDefense)); !ok || amt != 10 {
		t.Errorf("MAGIC_DEFENSE amount = %d (present=%v), want 10", amt, ok)
	}
}

// TestMountTamedPreservesStatupsAndOverridesVehicle verifies that a tamed mount
// keeps the skill's non-riding statups while overriding the MONSTER_RIDING
// amount with the equipped taming-mob id (slot -18).
func TestMountTamedPreservesStatupsAndOverridesVehicle(t *testing.T) {
	statups := []statup.RestModel{
		{Type: string(charconst.TemporaryStatTypeWeaponDefense), Amount: 15},
		{Type: string(charconst.TemporaryStatTypeMonsterRiding), Amount: int32(tamedMountSkillId)}, // atlas-data placeholder = skill id
	}
	d := &recordingDeps{
		mounted: false,
		equip:   map[int16]int32{-18: tamingMobItemId, -19: 1902020},
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(statups), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if !d.applyCalled {
		t.Fatalf("expected Apply called with both slots present")
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeMonsterRiding)); !ok || amt != tamingMobItemId {
		t.Errorf("MONSTER_RIDING amount = %d (present=%v), want taming-mob id %d", amt, ok, tamingMobItemId)
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeWeaponDefense)); !ok || amt != 15 {
		t.Errorf("WEAPON_DEFENSE amount = %d (present=%v), want 15", amt, ok)
	}
}

// TestMountTamedAppendsRidingWhenEffectLacksIt verifies the case-2 append branch:
// when the tamed mount's effect carries no MONSTER_RIDING statup, one is appended
// with the equipped taming-mob id while any other granted stats are preserved.
func TestMountTamedAppendsRidingWhenEffectLacksIt(t *testing.T) {
	statups := []statup.RestModel{
		{Type: string(charconst.TemporaryStatTypeWeaponDefense), Amount: 12}, // no MONSTER_RIDING entry
	}
	d := &recordingDeps{
		mounted: false,
		equip:   map[int16]int32{-18: tamingMobItemId, -19: 1902020},
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(statups), d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount returned error: %v", err)
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeMonsterRiding)); !ok || amt != tamingMobItemId {
		t.Errorf("MONSTER_RIDING amount = %d (present=%v), want appended taming-mob id %d", amt, ok, tamingMobItemId)
	}
	if amt, ok := findStatupAmount(d.applyStatups, string(charconst.TemporaryStatTypeWeaponDefense)); !ok || amt != 12 {
		t.Errorf("WEAPON_DEFENSE amount = %d (present=%v), want 12", amt, ok)
	}
}

var errStub = stubErr("slot read must not be called for skill-only mounts")

type stubErr string

func (e stubErr) Error() string { return string(e) }
