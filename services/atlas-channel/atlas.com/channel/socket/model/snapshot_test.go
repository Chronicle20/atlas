package model

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/equipment"
	equipmentslot "atlas-channel/equipment/slot"
	"atlas-channel/pet"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
)

// --- NewAssetSnapshot / NewAssetFromSnapshot round-trip (Step 1) ---

func TestAssetSnapshotRoundTrip_Equip(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	a, err := asset.NewModelBuilder(1, uuid.New(), 1302000).
		SetSlot(5).
		SetExpiration(expiration).
		SetStrength(10).
		SetDexterity(20).
		SetIntelligence(30).
		SetLuck(40).
		SetHp(50).
		SetMp(60).
		SetWeaponAttack(70).
		SetMagicAttack(80).
		SetWeaponDefense(90).
		SetMagicDefense(100).
		SetAccuracy(110).
		SetAvoidability(120).
		SetHands(130).
		SetSpeed(140).
		SetJump(150).
		SetSlots(7).
		SetLevelType(1).
		SetLevel(12).
		SetExperience(9999).
		SetHammersApplied(2).
		SetFlag(0x40).
		Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}

	snap := NewAssetSnapshot(a)
	out := NewAssetFromSnapshot(snap)

	if out.Slot() != a.Slot() {
		t.Errorf("Slot() = %d, want %d", out.Slot(), a.Slot())
	}
	if out.TemplateId() != a.TemplateId() {
		t.Errorf("TemplateId() = %d, want %d", out.TemplateId(), a.TemplateId())
	}
	if !out.Expiration().Equal(a.Expiration()) {
		t.Errorf("Expiration() = %v, want %v", out.Expiration(), a.Expiration())
	}
	if !out.IsEquipment() {
		t.Fatal("expected out.IsEquipment() true")
	}
	if out.Strength() != a.Strength() {
		t.Errorf("Strength() = %d, want %d", out.Strength(), a.Strength())
	}
	if out.Dexterity() != a.Dexterity() {
		t.Errorf("Dexterity() = %d, want %d", out.Dexterity(), a.Dexterity())
	}
	if out.Intelligence() != a.Intelligence() {
		t.Errorf("Intelligence() = %d, want %d", out.Intelligence(), a.Intelligence())
	}
	if out.Luck() != a.Luck() {
		t.Errorf("Luck() = %d, want %d", out.Luck(), a.Luck())
	}
	if out.Hp() != a.Hp() {
		t.Errorf("Hp() = %d, want %d", out.Hp(), a.Hp())
	}
	if out.Mp() != a.Mp() {
		t.Errorf("Mp() = %d, want %d", out.Mp(), a.Mp())
	}
	if out.WeaponAttack() != a.WeaponAttack() {
		t.Errorf("WeaponAttack() = %d, want %d", out.WeaponAttack(), a.WeaponAttack())
	}
	if out.MagicAttack() != a.MagicAttack() {
		t.Errorf("MagicAttack() = %d, want %d", out.MagicAttack(), a.MagicAttack())
	}
	if out.WeaponDefense() != a.WeaponDefense() {
		t.Errorf("WeaponDefense() = %d, want %d", out.WeaponDefense(), a.WeaponDefense())
	}
	if out.MagicDefense() != a.MagicDefense() {
		t.Errorf("MagicDefense() = %d, want %d", out.MagicDefense(), a.MagicDefense())
	}
	if out.Accuracy() != a.Accuracy() {
		t.Errorf("Accuracy() = %d, want %d", out.Accuracy(), a.Accuracy())
	}
	if out.Avoidability() != a.Avoidability() {
		t.Errorf("Avoidability() = %d, want %d", out.Avoidability(), a.Avoidability())
	}
	if out.Hands() != a.Hands() {
		t.Errorf("Hands() = %d, want %d", out.Hands(), a.Hands())
	}
	if out.Speed() != a.Speed() {
		t.Errorf("Speed() = %d, want %d", out.Speed(), a.Speed())
	}
	if out.Jump() != a.Jump() {
		t.Errorf("Jump() = %d, want %d", out.Jump(), a.Jump())
	}
	if out.Slots() != a.Slots() {
		t.Errorf("Slots() = %d, want %d", out.Slots(), a.Slots())
	}
	if out.LevelType() != a.LevelType() {
		t.Errorf("LevelType() = %d, want %d", out.LevelType(), a.LevelType())
	}
	if out.Level() != a.Level() {
		t.Errorf("Level() = %d, want %d", out.Level(), a.Level())
	}
	if out.Experience() != a.Experience() {
		t.Errorf("Experience() = %d, want %d", out.Experience(), a.Experience())
	}
	if out.HammersApplied() != a.HammersApplied() {
		t.Errorf("HammersApplied() = %d, want %d", out.HammersApplied(), a.HammersApplied())
	}
	if out.Flag() != a.Flag() {
		t.Errorf("Flag() = %d, want %d", out.Flag(), a.Flag())
	}
}

func TestAssetSnapshotRoundTrip_CashEquip(t *testing.T) {
	a, err := asset.NewModelBuilder(1, uuid.New(), 1302000).
		SetSlot(3).
		SetCashId(123456789).
		Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}

	snap := NewAssetSnapshot(a)
	out := NewAssetFromSnapshot(snap)

	if !out.IsCashEquipment() {
		t.Fatal("expected out.IsCashEquipment() true")
	}
	if out.CashId() != a.CashId() {
		t.Errorf("CashId() = %d, want %d", out.CashId(), a.CashId())
	}
}

func TestAssetSnapshotRoundTrip_Stackable(t *testing.T) {
	expiration := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	a, err := asset.NewModelBuilder(2, uuid.New(), 2000000).
		SetSlot(1).
		SetExpiration(expiration).
		SetQuantity(99).
		SetFlag(0x10).
		SetRechargeable(500).
		Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}

	snap := NewAssetSnapshot(a)
	out := NewAssetFromSnapshot(snap)

	if out.Slot() != a.Slot() {
		t.Errorf("Slot() = %d, want %d", out.Slot(), a.Slot())
	}
	if out.TemplateId() != a.TemplateId() {
		t.Errorf("TemplateId() = %d, want %d", out.TemplateId(), a.TemplateId())
	}
	if !out.Expiration().Equal(a.Expiration()) {
		t.Errorf("Expiration() = %v, want %v", out.Expiration(), a.Expiration())
	}
	if !out.IsConsumable() {
		t.Fatal("expected out.IsConsumable() true")
	}
	if out.Quantity() != a.Quantity() {
		t.Errorf("Quantity() = %d, want %d", out.Quantity(), a.Quantity())
	}
	if out.Flag() != a.Flag() {
		t.Errorf("Flag() = %d, want %d", out.Flag(), a.Flag())
	}
	if out.Rechargeable() != a.Rechargeable() {
		t.Errorf("Rechargeable() = %d, want %d", out.Rechargeable(), a.Rechargeable())
	}
}

func TestAssetSnapshotRoundTrip_Pet(t *testing.T) {
	expiration := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	a, err := asset.NewModelBuilder(3, uuid.New(), 5000000).
		SetSlot(0).
		SetExpiration(expiration).
		SetPetId(777).
		SetPetName("Fluffy").
		SetPetLevel(15).
		SetCloseness(9000).
		SetFullness(88).
		Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}
	if !a.IsPet() {
		t.Fatal("test setup: expected a.IsPet() true")
	}

	snap := NewAssetSnapshot(a)
	out := NewAssetFromSnapshot(snap)

	if out.TemplateId() != a.TemplateId() {
		t.Errorf("TemplateId() = %d, want %d", out.TemplateId(), a.TemplateId())
	}
	if !out.IsPet() {
		t.Fatal("expected out.IsPet() true")
	}
	if out.PetId() != a.PetId() {
		t.Errorf("PetId() = %d, want %d", out.PetId(), a.PetId())
	}
	if out.PetName() != a.PetName() {
		t.Errorf("PetName() = %q, want %q", out.PetName(), a.PetName())
	}
	if out.PetLevel() != a.PetLevel() {
		t.Errorf("PetLevel() = %d, want %d", out.PetLevel(), a.PetLevel())
	}
	if out.Closeness() != a.Closeness() {
		t.Errorf("Closeness() = %d, want %d", out.Closeness(), a.Closeness())
	}
	if out.Fullness() != a.Fullness() {
		t.Errorf("Fullness() = %d, want %d", out.Fullness(), a.Fullness())
	}
	if !out.Expiration().Equal(a.Expiration()) {
		t.Errorf("Expiration() = %v, want %v", out.Expiration(), a.Expiration())
	}
}

// --- NewAvatarSnapshot / NewAvatarFromSnapshot round-trip (Step 1) ---

func buildTestEquipment(t *testing.T) equipment.Model {
	t.Helper()
	eq := equipment.NewModel()

	weapon := asset.NewModelBuilder(10, uuid.New(), 1302001).MustBuild()
	eq.Set("weapon", equipmentslot.Model{Position: -11, Equipable: &weapon})

	hat := asset.NewModelBuilder(11, uuid.New(), 1002001).MustBuild()
	eq.Set("hat", equipmentslot.Model{Position: -1, Equipable: &hat})

	// A masked cash equip: the CashEquipable renders as the equipped look and
	// the underlying Equipable renders separately into MaskedEquips, mirroring
	// NewFromCharacter (avatar.go:14-32).
	cashTop := asset.NewModelBuilder(12, uuid.New(), 1052001).SetCashId(55).MustBuild()
	realTop := asset.NewModelBuilder(13, uuid.New(), 1052002).MustBuild()
	eq.Set("top", equipmentslot.Model{Position: -5, Equipable: &realTop, CashEquipable: &cashTop})

	return eq
}

func TestAvatarSnapshotRoundTrip(t *testing.T) {
	eq := buildTestEquipment(t)
	pets := []pet.Model{
		pet.NewModelBuilder(1, 0, 5000028, "Kitty").SetSlot(0).MustBuild(),
		pet.NewModelBuilder(2, 0, 5000029, "Puppy").SetSlot(1).MustBuild(),
	}

	c, err := character.NewModelBuilder().
		SetId(1).
		SetGender(1).
		SetSkinColor(3).
		SetFace(20000).
		SetHair(30000).
		SetEquipment(eq).
		SetPets(pets).
		Build()
	if err != nil {
		t.Fatalf("character build: %v", err)
	}

	snap := NewAvatarSnapshot(c)

	if snap.Gender != c.Gender() {
		t.Errorf("Gender = %d, want %d", snap.Gender, c.Gender())
	}
	if snap.SkinColor != c.SkinColor() {
		t.Errorf("SkinColor = %d, want %d", snap.SkinColor, c.SkinColor())
	}
	if snap.Face != c.Face() {
		t.Errorf("Face = %d, want %d", snap.Face, c.Face())
	}
	if snap.Hair != c.Hair() {
		t.Errorf("Hair = %d, want %d", snap.Hair, c.Hair())
	}
	// weapon (-11 -> key 11), hat (-1 -> key 1): rendered into Equips.
	if got := snap.Equips[11]; got != 1302001 {
		t.Errorf("Equips[11] = %d, want 1302001 (weapon)", got)
	}
	if got := snap.Equips[1]; got != 1002001 {
		t.Errorf("Equips[1] = %d, want 1002001 (hat)", got)
	}
	// masked cash top (-5 -> key 5): cash template rendered into Equips,
	// underlying real template rendered into MaskedEquips.
	if got := snap.Equips[5]; got != 1052001 {
		t.Errorf("Equips[5] = %d, want 1052001 (cash top)", got)
	}
	if got := snap.MaskedEquips[5]; got != 1052002 {
		t.Errorf("MaskedEquips[5] = %d, want 1052002 (masked real top)", got)
	}
	if got := snap.Pets[0]; got != 5000028 {
		t.Errorf("Pets[0] = %d, want 5000028", got)
	}
	if got := snap.Pets[1]; got != 5000029 {
		t.Errorf("Pets[1] = %d, want 5000029", got)
	}

	out := NewAvatarFromSnapshot(snap, true)

	if out.Gender() != c.Gender() {
		t.Errorf("out.Gender() = %d, want %d", out.Gender(), c.Gender())
	}
	if out.SkinColor() != c.SkinColor() {
		t.Errorf("out.SkinColor() = %d, want %d", out.SkinColor(), c.SkinColor())
	}
	if out.Face() != c.Face() {
		t.Errorf("out.Face() = %d, want %d", out.Face(), c.Face())
	}
	if out.Hair() != c.Hair() {
		t.Errorf("out.Hair() = %d, want %d", out.Hair(), c.Hair())
	}
	if !out.Mega() {
		t.Error("expected out.Mega() true")
	}
	if len(out.Equipment()) != len(snap.Equips) {
		t.Errorf("len(out.Equipment()) = %d, want %d", len(out.Equipment()), len(snap.Equips))
	}
	for k, v := range snap.Equips {
		if got := out.Equipment()[slot.Position(k)]; got != v {
			t.Errorf("out.Equipment()[%d] = %d, want %d", k, got, v)
		}
	}
	if len(out.MaskedEquipment()) != len(snap.MaskedEquips) {
		t.Errorf("len(out.MaskedEquipment()) = %d, want %d", len(out.MaskedEquipment()), len(snap.MaskedEquips))
	}
	for k, v := range snap.MaskedEquips {
		if got := out.MaskedEquipment()[slot.Position(k)]; got != v {
			t.Errorf("out.MaskedEquipment()[%d] = %d, want %d", k, got, v)
		}
	}
	if len(out.Pets()) != len(snap.Pets) {
		t.Errorf("len(out.Pets()) = %d, want %d", len(out.Pets()), len(snap.Pets))
	}
	for k, v := range snap.Pets {
		if got := out.Pets()[k]; got != v {
			t.Errorf("out.Pets()[%d] = %d, want %d", k, got, v)
		}
	}
}
