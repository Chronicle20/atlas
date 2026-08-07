package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	tamedMountSkillId     = uint32(skill2.BeginnerMonsterRidingId) // 1004
	skillOnlyMountSkillId = uint32(skill2.BeginnerBroomstickId)    // 1019 -> vehicle 1932005
	tamingMobItemId       = int32(1902000)                         // arbitrary equipped taming-mob id
)

// tamedMountIdentity / skillOnlyMountIdentity are the Identity-typed
// counterparts HandleMount now takes directly (task-187): production
// resolves the caster's wire skill id through the tenant's version set
// before calling in, but these mount roots are version-stable, so the
// canonical wire id and Identity token coincide numerically and tests can
// pass the identity constant directly without a tenant context.
const (
	tamedMountIdentity     = skill2.BeginnerMonsterRiding
	skillOnlyMountIdentity = skill2.BeginnerBroomstick
	battleshipIdentity     = skill2.CorsairBattleship
)

// recordingDeps captures collaborator invocations so each of the five mount
// cases can be asserted offline without Kafka, REST, or a session.
type recordingDeps struct {
	mounted        bool
	mountedErr     error
	equip          map[int16]int32 // slot position -> taming-mob/saddle item id
	equipErr       map[int16]error
	applyCalled    bool
	applyAmount    int32
	applyStatups   []statup.Model
	applySource    int32
	cancelCount    int
	cancelSrc      int32
	vehicleId      int32
	vehicleOk      bool
	charLevel      byte
	charLevelErr   error
	initCalled     bool
	initSkillLevel byte
	initCharLevel  byte
	initTTL        time.Duration
	initErr        error
	clearCalled    bool
	clearCount     int
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
		applyBuff: func(f field.Model, characterId uint32, sourceId int32, level byte, statups []statup.Model) error {
			d.applyCalled = true
			d.applySource = sourceId
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
		resolveVehicleId: func() (int32, bool) {
			return d.vehicleId, d.vehicleOk
		},
		characterLevel: func(characterId uint32) (byte, error) {
			return d.charLevel, d.charLevelErr
		},
		initShipHP: func(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error {
			d.initCalled = true
			d.initSkillLevel = skillLevel
			d.initCharLevel = charLevel
			d.initTTL = ttl
			return d.initErr
		},
		clearShipHP: func(characterId uint32) {
			d.clearCalled = true
			d.clearCount++
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), tamedMountIdentity, d.mountDeps())
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), tamedMountIdentity, d.mountDeps())
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), tamedMountIdentity, d.mountDeps())
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
	if d.cancelCount != 0 {
		t.Errorf("expected no Cancel, got %d", d.cancelCount)
	}
}

func TestMountTamedSlot18EmptyNoOp(t *testing.T) {
	d := &recordingDeps{
		mounted: false,
		equip:   map[int16]int32{-19: 1902020}, // -18 empty
	}
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(nil), tamedMountIdentity, d.mountDeps())
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(skillOnlyMountSkillId), mountEffect(vehicleStatup(vehicleId)), skillOnlyMountIdentity, d.mountDeps())
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(skillOnlyMountSkillId), mountEffect(statups), skillOnlyMountIdentity, d.mountDeps())
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(statups), tamedMountIdentity, d.mountDeps())
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
	err := HandleMount(logrus.New(), field.Model{}, 100, mountInfo(tamedMountSkillId), mountEffect(statups), tamedMountIdentity, d.mountDeps())
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

const battleshipSkillId = uint32(skill2.CorsairBattleshipId)

// battleshipInfo mirrors mountInfo (mount_test.go:73) but with a settable level.
func battleshipInfo(level byte) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().SetSkillId(battleshipSkillId).SetSkillLevel(level).Build()
}

// battleshipEffect mirrors mountEffect (mount_test.go:77): Duration 2100000 ms
// (the WZ buff time) and a MONSTER_RIDING statup carrying atlas-data's
// skill-id placeholder amount, which the arm must override.
func battleshipEffect() effect.Model {
	e, err := effect.Extract(effect.RestModel{Statups: vehicleStatup(int32(battleshipSkillId)), Duration: 2100000})
	if err != nil {
		panic(err)
	}
	return e
}

func TestHandleMountBattleshipApplies(t *testing.T) {
	d := &recordingDeps{vehicleId: 1932000, vehicleOk: true, charLevel: 150}

	if err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), battleshipIdentity, d.mountDeps()); err != nil {
		t.Fatalf("HandleMount: %v", err)
	}
	if !d.applyCalled {
		t.Fatal("expected applyBuff to be called")
	}
	if d.applyAmount != 1932000 {
		t.Fatalf("MONSTER_RIDING amount = %d, want config-resolved 1932000", d.applyAmount)
	}
	if !d.initCalled || d.initSkillLevel != 7 || d.initCharLevel != 150 {
		t.Fatalf("initShipHP = (called %v, skill %d, char %d), want (true, 7, 150)", d.initCalled, d.initSkillLevel, d.initCharLevel)
	}
	if d.initTTL != 2100000*time.Millisecond {
		t.Fatalf("init TTL = %v, want 35m from effect duration", d.initTTL)
	}
}

func TestHandleMountBattleshipAbortsOnVehicleMiss(t *testing.T) {
	d := &recordingDeps{vehicleOk: false, charLevel: 150}
	if err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), battleshipIdentity, d.mountDeps()); err != nil {
		t.Fatalf("HandleMount: %v", err)
	}
	if d.applyCalled || d.initCalled {
		t.Fatal("resolve miss must abort: no buff, no HP state")
	}
}

func TestHandleMountBattleshipToggleDismounts(t *testing.T) {
	d := &recordingDeps{mounted: true, vehicleId: 1932000, vehicleOk: true, charLevel: 150}
	if err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), battleshipIdentity, d.mountDeps()); err != nil {
		t.Fatalf("HandleMount: %v", err)
	}
	if d.cancelCount != 1 || d.applyCalled || d.initCalled {
		t.Fatalf("toggle must only cancel: cancels %d, applied %v, init %v", d.cancelCount, d.applyCalled, d.initCalled)
	}
}

// TestHandleMountBattleshipAbortsOnCharacterLevelError verifies that a
// characterLevel failure aborts the mount entirely: the error propagates,
// and neither the buff nor the ship HP pool is touched.
func TestHandleMountBattleshipAbortsOnCharacterLevelError(t *testing.T) {
	d := &recordingDeps{vehicleId: 1932000, vehicleOk: true, charLevelErr: errCharacterLevelStub}
	err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), battleshipIdentity, d.mountDeps())
	if err == nil {
		t.Fatal("expected characterLevel error to propagate")
	}
	if d.applyCalled || d.initCalled {
		t.Fatal("characterLevel failure must abort before Apply/initShipHP")
	}
}

// TestHandleMountBattleshipInitFailureClearsStalePool verifies the Finding-1
// fix: an initShipHP failure must not abort the mount (Redis trouble never
// blocks a mount) but MUST best-effort clear any stale pool from a prior
// ride, so the next Drain takes the lazy full re-init path instead of
// decrementing leftover HP from before this mount.
func TestHandleMountBattleshipInitFailureClearsStalePool(t *testing.T) {
	d := &recordingDeps{vehicleId: 1932000, vehicleOk: true, charLevel: 150, initErr: errInitShipHPStub}
	err := HandleMount(logrus.New(), field.Model{}, 999, battleshipInfo(7), battleshipEffect(), battleshipIdentity, d.mountDeps())
	if err != nil {
		t.Fatalf("HandleMount: %v (initShipHP failure must not abort the mount)", err)
	}
	if !d.applyCalled {
		t.Fatal("expected applyBuff to still be called despite initShipHP failure")
	}
	if !d.clearCalled || d.clearCount != 1 {
		t.Fatalf("expected clearShipHP called exactly once, got called=%v count=%d", d.clearCalled, d.clearCount)
	}
}

var errStub = stubErr("slot read must not be called for skill-only mounts")

var errCharacterLevelStub = stubErr("character level load failed")

var errInitShipHPStub = stubErr("ship HP init failed (e.g. transient Redis error)")

type stubErr string

func (e stubErr) Error() string { return string(e) }
