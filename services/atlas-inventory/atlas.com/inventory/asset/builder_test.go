package asset_test

import (
	"atlas-inventory/asset"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewBuilder(t *testing.T) {
	compartmentId := uuid.New()
	templateId := uint32(1040010)

	b := asset.NewBuilder(compartmentId, templateId)
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}

	m := b.Build()
	if m.CompartmentId() != compartmentId {
		t.Errorf("expected CompartmentId %s, got %s", compartmentId, m.CompartmentId())
	}
	if m.TemplateId() != templateId {
		t.Errorf("expected TemplateId %d, got %d", templateId, m.TemplateId())
	}
	if m.Slot() != 0 {
		t.Errorf("expected default Slot 0, got %d", m.Slot())
	}
	if !m.Expiration().IsZero() {
		t.Errorf("expected zero Expiration, got %v", m.Expiration())
	}
}

func TestBuilderSetSlot(t *testing.T) {
	compartmentId := uuid.New()
	m := asset.NewBuilder(compartmentId, 1040010).
		SetSlot(5).
		Build()

	if m.Slot() != 5 {
		t.Errorf("expected Slot 5, got %d", m.Slot())
	}
}

func TestBuilderSetExpiration(t *testing.T) {
	compartmentId := uuid.New()
	expTime := time.Now().Add(24 * time.Hour)

	m := asset.NewBuilder(compartmentId, 1040010).
		SetExpiration(expTime).
		Build()

	if !m.Expiration().Equal(expTime) {
		t.Errorf("expected Expiration %v, got %v", expTime, m.Expiration())
	}
}

func TestBuilderEquipmentFields(t *testing.T) {
	compartmentId := uuid.New()

	m := asset.NewBuilder(compartmentId, 1040010).
		SetStrength(10).
		SetDexterity(5).
		SetIntelligence(3).
		SetLuck(7).
		SetHp(100).
		SetMp(50).
		SetWeaponAttack(15).
		SetMagicAttack(12).
		SetWeaponDefense(8).
		SetMagicDefense(6).
		SetAccuracy(4).
		SetAvoidability(2).
		SetHands(1).
		SetSpeed(3).
		SetJump(2).
		SetSlots(7).
		Build()

	if m.Strength() != 10 {
		t.Errorf("expected Strength 10, got %d", m.Strength())
	}
	if m.Dexterity() != 5 {
		t.Errorf("expected Dexterity 5, got %d", m.Dexterity())
	}
	if m.Intelligence() != 3 {
		t.Errorf("expected Intelligence 3, got %d", m.Intelligence())
	}
	if m.Luck() != 7 {
		t.Errorf("expected Luck 7, got %d", m.Luck())
	}
	if m.Hp() != 100 {
		t.Errorf("expected HP 100, got %d", m.Hp())
	}
	if m.Mp() != 50 {
		t.Errorf("expected MP 50, got %d", m.Mp())
	}
	if m.WeaponAttack() != 15 {
		t.Errorf("expected WeaponAttack 15, got %d", m.WeaponAttack())
	}
	if m.MagicAttack() != 12 {
		t.Errorf("expected MagicAttack 12, got %d", m.MagicAttack())
	}
	if m.WeaponDefense() != 8 {
		t.Errorf("expected WeaponDefense 8, got %d", m.WeaponDefense())
	}
	if m.MagicDefense() != 6 {
		t.Errorf("expected MagicDefense 6, got %d", m.MagicDefense())
	}
	if m.Accuracy() != 4 {
		t.Errorf("expected Accuracy 4, got %d", m.Accuracy())
	}
	if m.Avoidability() != 2 {
		t.Errorf("expected Avoidability 2, got %d", m.Avoidability())
	}
	if m.Hands() != 1 {
		t.Errorf("expected Hands 1, got %d", m.Hands())
	}
	if m.Speed() != 3 {
		t.Errorf("expected Speed 3, got %d", m.Speed())
	}
	if m.Jump() != 2 {
		t.Errorf("expected Jump 2, got %d", m.Jump())
	}
	if m.Slots() != 7 {
		t.Errorf("expected Slots 7, got %d", m.Slots())
	}
}

func TestBuilderStackableFields(t *testing.T) {
	compartmentId := uuid.New()

	m := asset.NewBuilder(compartmentId, 2000100).
		SetQuantity(50).
		SetOwnerId(123).
		SetFlag(1).
		SetRechargeable(100).
		Build()

	if m.Rechargeable() != 100 {
		t.Errorf("expected Rechargeable 100, got %d", m.Rechargeable())
	}
	if m.OwnerId() != 123 {
		t.Errorf("expected OwnerId 123, got %d", m.OwnerId())
	}
	if m.Flag() != 1 {
		t.Errorf("expected Flag 1, got %d", m.Flag())
	}
}

func TestClone(t *testing.T) {
	compartmentId := uuid.New()
	expTime := time.Now()

	original := asset.NewBuilder(compartmentId, 1040010).
		SetId(1).
		SetSlot(3).
		SetExpiration(expTime).
		SetStrength(10).
		SetWeaponDefense(5).
		SetSlots(7).
		Build()

	cloned := asset.Clone(original).Build()

	if cloned.Id() != original.Id() {
		t.Errorf("cloned Id %d != original Id %d", cloned.Id(), original.Id())
	}
	if cloned.CompartmentId() != original.CompartmentId() {
		t.Errorf("cloned CompartmentId %s != original CompartmentId %s", cloned.CompartmentId(), original.CompartmentId())
	}
	if cloned.TemplateId() != original.TemplateId() {
		t.Errorf("cloned TemplateId %d != original TemplateId %d", cloned.TemplateId(), original.TemplateId())
	}
	if cloned.Slot() != original.Slot() {
		t.Errorf("cloned Slot %d != original Slot %d", cloned.Slot(), original.Slot())
	}
	if !cloned.Expiration().Equal(original.Expiration()) {
		t.Errorf("cloned Expiration %v != original Expiration %v", cloned.Expiration(), original.Expiration())
	}
	if cloned.Strength() != original.Strength() {
		t.Errorf("cloned Strength %d != original Strength %d", cloned.Strength(), original.Strength())
	}
	if cloned.WeaponDefense() != original.WeaponDefense() {
		t.Errorf("cloned WeaponDefense %d != original WeaponDefense %d", cloned.WeaponDefense(), original.WeaponDefense())
	}
	if cloned.Slots() != original.Slots() {
		t.Errorf("cloned Slots %d != original Slots %d", cloned.Slots(), original.Slots())
	}
}

func TestCloneAndModify(t *testing.T) {
	compartmentId := uuid.New()
	original := asset.NewBuilder(compartmentId, 1040010).
		SetSlot(1).
		Build()

	modified := asset.Clone(original).
		SetSlot(5).
		Build()

	if original.Slot() != 1 {
		t.Errorf("original Slot changed: expected 1, got %d", original.Slot())
	}
	if modified.Slot() != 5 {
		t.Errorf("modified Slot incorrect: expected 5, got %d", modified.Slot())
	}
}

func TestFluentChaining(t *testing.T) {
	compartmentId := uuid.New()
	b := asset.NewBuilder(compartmentId, 1040010)

	result := b.SetSlot(1)
	if result != b {
		t.Error("SetSlot did not return the builder")
	}

	result = b.SetExpiration(time.Now())
	if result != b {
		t.Error("SetExpiration did not return the builder")
	}

	result = b.SetQuantity(10)
	if result != b {
		t.Error("SetQuantity did not return the builder")
	}
}

func TestInventoryTypeDerivation(t *testing.T) {
	compartmentId := uuid.New()

	// Equip item (1000000 range)
	equipModel := asset.NewBuilder(compartmentId, 1040010).Build()
	if !equipModel.IsEquipment() {
		t.Error("expected IsEquipment true for templateId 1040010")
	}

	// Use item (2000000 range)
	useModel := asset.NewBuilder(compartmentId, 2000100).Build()
	if !useModel.IsConsumable() {
		t.Error("expected IsConsumable true for templateId 2000100")
	}

	// Setup item (3000000 range)
	setupModel := asset.NewBuilder(compartmentId, 3010000).Build()
	if !setupModel.IsSetup() {
		t.Error("expected IsSetup true for templateId 3010000")
	}

	// Etc item (4000000 range)
	etcModel := asset.NewBuilder(compartmentId, 4000100).Build()
	if !etcModel.IsEtc() {
		t.Error("expected IsEtc true for templateId 4000100")
	}

	// Cash item (5000000 range)
	cashModel := asset.NewBuilder(compartmentId, 5000100).Build()
	if !cashModel.IsCash() {
		t.Error("expected IsCash true for templateId 5000100")
	}
}

func TestNegativeSlot(t *testing.T) {
	compartmentId := uuid.New()
	m := asset.NewBuilder(compartmentId, 1040010).
		SetSlot(-5).
		Build()

	if m.Slot() != -5 {
		t.Errorf("expected negative Slot -5, got %d", m.Slot())
	}
}
