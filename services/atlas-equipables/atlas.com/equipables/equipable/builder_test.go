package equipable_test

import (
	"atlas-equipables/equipable"
	"testing"
	"time"
)

func TestNewBuilderSetsId(t *testing.T) {
	builder := equipable.NewBuilder(12345)
	model := builder.Build()

	if model.Id() != 12345 {
		t.Fatalf("Expected ID to be 12345, was %d", model.Id())
	}
}

func TestBuilderFluentMethods(t *testing.T) {
	// Verify all setter methods return the builder for chaining
	builder := equipable.NewBuilder(1)

	result := builder.
		SetItemId(100).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetHp(100).
		SetMp(50).
		SetWeaponAttack(20).
		SetMagicAttack(15).
		SetWeaponDefense(25).
		SetMagicDefense(20).
		SetAccuracy(5).
		SetAvoidability(5).
		SetHands(2).
		SetSpeed(10).
		SetJump(5).
		SetSlots(7).
		SetOwnerName("TestOwner").
		SetLocked(true).
		SetSpikes(true).
		SetKarmaUsed(true).
		SetCold(true).
		SetCanBeTraded(false).
		SetLevelType(1).
		SetLevel(5).
		SetExperience(1000).
		SetHammersApplied(3).
		SetExpiration(time.Now())

	// If we got here without panic, chaining works
	if result == nil {
		t.Fatal("Builder chaining should return non-nil builder")
	}
}

func TestBuilderBuild(t *testing.T) {
	expTime := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	model := equipable.NewBuilder(100).
		SetItemId(1302000).
		SetStrength(50).
		SetDexterity(40).
		SetIntelligence(30).
		SetLuck(20).
		SetHp(500).
		SetMp(300).
		SetWeaponAttack(100).
		SetMagicAttack(80).
		SetWeaponDefense(60).
		SetMagicDefense(40).
		SetAccuracy(20).
		SetAvoidability(15).
		SetHands(10).
		SetSpeed(25).
		SetJump(15).
		SetSlots(7).
		SetOwnerName("Builder").
		SetLocked(true).
		SetSpikes(false).
		SetKarmaUsed(true).
		SetCold(false).
		SetCanBeTraded(true).
		SetLevelType(2).
		SetLevel(10).
		SetExperience(5000).
		SetHammersApplied(5).
		SetExpiration(expTime).
		Build()

	// Verify all fields
	if model.Id() != 100 {
		t.Fatalf("Id should be 100, was %d", model.Id())
	}
	if model.ItemId() != 1302000 {
		t.Fatalf("ItemId should be 1302000, was %d", model.ItemId())
	}
	if model.Strength() != 50 {
		t.Fatalf("Strength should be 50, was %d", model.Strength())
	}
	if model.Dexterity() != 40 {
		t.Fatalf("Dexterity should be 40, was %d", model.Dexterity())
	}
	if model.Intelligence() != 30 {
		t.Fatalf("Intelligence should be 30, was %d", model.Intelligence())
	}
	if model.Luck() != 20 {
		t.Fatalf("Luck should be 20, was %d", model.Luck())
	}
	if model.HP() != 500 {
		t.Fatalf("HP should be 500, was %d", model.HP())
	}
	if model.MP() != 300 {
		t.Fatalf("MP should be 300, was %d", model.MP())
	}
	if model.WeaponAttack() != 100 {
		t.Fatalf("WeaponAttack should be 100, was %d", model.WeaponAttack())
	}
	if model.MagicAttack() != 80 {
		t.Fatalf("MagicAttack should be 80, was %d", model.MagicAttack())
	}
	if model.WeaponDefense() != 60 {
		t.Fatalf("WeaponDefense should be 60, was %d", model.WeaponDefense())
	}
	if model.MagicDefense() != 40 {
		t.Fatalf("MagicDefense should be 40, was %d", model.MagicDefense())
	}
	if model.Accuracy() != 20 {
		t.Fatalf("Accuracy should be 20, was %d", model.Accuracy())
	}
	if model.Avoidability() != 15 {
		t.Fatalf("Avoidability should be 15, was %d", model.Avoidability())
	}
	if model.Hands() != 10 {
		t.Fatalf("Hands should be 10, was %d", model.Hands())
	}
	if model.Speed() != 25 {
		t.Fatalf("Speed should be 25, was %d", model.Speed())
	}
	if model.Jump() != 15 {
		t.Fatalf("Jump should be 15, was %d", model.Jump())
	}
	if model.Slots() != 7 {
		t.Fatalf("Slots should be 7, was %d", model.Slots())
	}
	if model.OwnerName() != "Builder" {
		t.Fatalf("OwnerName should be 'Builder', was '%s'", model.OwnerName())
	}
	if !model.Locked() {
		t.Fatal("Locked should be true")
	}
	if model.Spikes() {
		t.Fatal("Spikes should be false")
	}
	if !model.KarmaUsed() {
		t.Fatal("KarmaUsed should be true")
	}
	if model.Cold() {
		t.Fatal("Cold should be false")
	}
	if !model.CanBeTraded() {
		t.Fatal("CanBeTraded should be true")
	}
	if model.LevelType() != 2 {
		t.Fatalf("LevelType should be 2, was %d", model.LevelType())
	}
	if model.Level() != 10 {
		t.Fatalf("Level should be 10, was %d", model.Level())
	}
	if model.Experience() != 5000 {
		t.Fatalf("Experience should be 5000, was %d", model.Experience())
	}
	if model.HammersApplied() != 5 {
		t.Fatalf("HammersApplied should be 5, was %d", model.HammersApplied())
	}
	if !model.Expiration().Equal(expTime) {
		t.Fatalf("Expiration should be %v, was %v", expTime, model.Expiration())
	}
}

func TestCloneCreatesBuilderFromModel(t *testing.T) {
	expTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	original := equipable.NewBuilder(200).
		SetItemId(1402000).
		SetStrength(100).
		SetDexterity(80).
		SetIntelligence(60).
		SetLuck(40).
		SetHp(1000).
		SetMp(500).
		SetWeaponAttack(150).
		SetMagicAttack(120).
		SetWeaponDefense(90).
		SetMagicDefense(70).
		SetAccuracy(30).
		SetAvoidability(25).
		SetHands(15).
		SetSpeed(35).
		SetJump(20).
		SetSlots(5).
		SetOwnerName("Original").
		SetLocked(false).
		SetSpikes(true).
		SetKarmaUsed(false).
		SetCold(true).
		SetCanBeTraded(false).
		SetLevelType(3).
		SetLevel(15).
		SetExperience(10000).
		SetHammersApplied(7).
		SetExpiration(expTime).
		Build()

	cloned := equipable.Clone(original).Build()

	// Verify all fields are preserved
	if cloned.Id() != original.Id() {
		t.Fatalf("Cloned Id should match original: expected %d, got %d", original.Id(), cloned.Id())
	}
	if cloned.ItemId() != original.ItemId() {
		t.Fatalf("Cloned ItemId should match original")
	}
	if cloned.Strength() != original.Strength() {
		t.Fatalf("Cloned Strength should match original")
	}
	if cloned.Dexterity() != original.Dexterity() {
		t.Fatalf("Cloned Dexterity should match original")
	}
	if cloned.Intelligence() != original.Intelligence() {
		t.Fatalf("Cloned Intelligence should match original")
	}
	if cloned.Luck() != original.Luck() {
		t.Fatalf("Cloned Luck should match original")
	}
	if cloned.HP() != original.HP() {
		t.Fatalf("Cloned HP should match original")
	}
	if cloned.MP() != original.MP() {
		t.Fatalf("Cloned MP should match original")
	}
	if cloned.WeaponAttack() != original.WeaponAttack() {
		t.Fatalf("Cloned WeaponAttack should match original")
	}
	if cloned.MagicAttack() != original.MagicAttack() {
		t.Fatalf("Cloned MagicAttack should match original")
	}
	if cloned.WeaponDefense() != original.WeaponDefense() {
		t.Fatalf("Cloned WeaponDefense should match original")
	}
	if cloned.MagicDefense() != original.MagicDefense() {
		t.Fatalf("Cloned MagicDefense should match original")
	}
	if cloned.Accuracy() != original.Accuracy() {
		t.Fatalf("Cloned Accuracy should match original")
	}
	if cloned.Avoidability() != original.Avoidability() {
		t.Fatalf("Cloned Avoidability should match original")
	}
	if cloned.Hands() != original.Hands() {
		t.Fatalf("Cloned Hands should match original")
	}
	if cloned.Speed() != original.Speed() {
		t.Fatalf("Cloned Speed should match original")
	}
	if cloned.Jump() != original.Jump() {
		t.Fatalf("Cloned Jump should match original")
	}
	if cloned.Slots() != original.Slots() {
		t.Fatalf("Cloned Slots should match original")
	}
	if cloned.OwnerName() != original.OwnerName() {
		t.Fatalf("Cloned OwnerName should match original")
	}
	if cloned.Locked() != original.Locked() {
		t.Fatalf("Cloned Locked should match original")
	}
	if cloned.Spikes() != original.Spikes() {
		t.Fatalf("Cloned Spikes should match original")
	}
	if cloned.KarmaUsed() != original.KarmaUsed() {
		t.Fatalf("Cloned KarmaUsed should match original")
	}
	if cloned.Cold() != original.Cold() {
		t.Fatalf("Cloned Cold should match original")
	}
	if cloned.CanBeTraded() != original.CanBeTraded() {
		t.Fatalf("Cloned CanBeTraded should match original")
	}
	if cloned.LevelType() != original.LevelType() {
		t.Fatalf("Cloned LevelType should match original")
	}
	if cloned.Level() != original.Level() {
		t.Fatalf("Cloned Level should match original")
	}
	if cloned.Experience() != original.Experience() {
		t.Fatalf("Cloned Experience should match original")
	}
	if cloned.HammersApplied() != original.HammersApplied() {
		t.Fatalf("Cloned HammersApplied should match original")
	}
	if !cloned.Expiration().Equal(original.Expiration()) {
		t.Fatalf("Cloned Expiration should match original")
	}
}

func TestCloneAllowsModification(t *testing.T) {
	original := equipable.NewBuilder(300).
		SetItemId(1502000).
		SetStrength(50).
		SetDexterity(50).
		Build()

	modified := equipable.Clone(original).
		SetStrength(100).
		SetDexterity(100).
		Build()

	// Original should be unchanged
	if original.Strength() != 50 {
		t.Fatalf("Original strength should be 50, was %d", original.Strength())
	}
	if original.Dexterity() != 50 {
		t.Fatalf("Original dexterity should be 50, was %d", original.Dexterity())
	}

	// Modified should have new values
	if modified.Strength() != 100 {
		t.Fatalf("Modified strength should be 100, was %d", modified.Strength())
	}
	if modified.Dexterity() != 100 {
		t.Fatalf("Modified dexterity should be 100, was %d", modified.Dexterity())
	}

	// ID should be preserved
	if modified.Id() != original.Id() {
		t.Fatalf("Modified ID should match original: expected %d, got %d", original.Id(), modified.Id())
	}
}

func TestAddStrengthClampsAtZero(t *testing.T) {
	builder := equipable.NewBuilder(1).SetStrength(50)

	// Adding -100 to 50 should clamp at 0, not go negative
	model := builder.AddStrength(-100).Build()

	if model.Strength() != 0 {
		t.Fatalf("Strength should clamp at 0, was %d", model.Strength())
	}
}

func TestAddStrengthClampsAtMax(t *testing.T) {
	builder := equipable.NewBuilder(1).SetStrength(65530)

	// Adding 100 to 65530 should clamp at uint16 max (65535)
	model := builder.AddStrength(100).Build()

	if model.Strength() != 65535 {
		t.Fatalf("Strength should clamp at 65535, was %d", model.Strength())
	}
}

func TestAddDexterityClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetDexterity(30).AddDexterity(-50).Build()
	if model.Dexterity() != 0 {
		t.Fatalf("Dexterity should clamp at 0, was %d", model.Dexterity())
	}
}

func TestAddDexterityClampsAtMax(t *testing.T) {
	model := equipable.NewBuilder(1).SetDexterity(65500).AddDexterity(100).Build()
	if model.Dexterity() != 65535 {
		t.Fatalf("Dexterity should clamp at 65535, was %d", model.Dexterity())
	}
}

func TestAddIntelligenceClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetIntelligence(20).AddIntelligence(-30).Build()
	if model.Intelligence() != 0 {
		t.Fatalf("Intelligence should clamp at 0, was %d", model.Intelligence())
	}
}

func TestAddLuckClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetLuck(10).AddLuck(-20).Build()
	if model.Luck() != 0 {
		t.Fatalf("Luck should clamp at 0, was %d", model.Luck())
	}
}

func TestAddHpClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetHp(100).AddHp(-200).Build()
	if model.HP() != 0 {
		t.Fatalf("HP should clamp at 0, was %d", model.HP())
	}
}

func TestAddMpClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetMp(50).AddMp(-100).Build()
	if model.MP() != 0 {
		t.Fatalf("MP should clamp at 0, was %d", model.MP())
	}
}

func TestAddWeaponAttackClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetWeaponAttack(25).AddWeaponAttack(-50).Build()
	if model.WeaponAttack() != 0 {
		t.Fatalf("WeaponAttack should clamp at 0, was %d", model.WeaponAttack())
	}
}

func TestAddMagicAttackClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetMagicAttack(25).AddMagicAttack(-50).Build()
	if model.MagicAttack() != 0 {
		t.Fatalf("MagicAttack should clamp at 0, was %d", model.MagicAttack())
	}
}

func TestAddSlotsClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetSlots(5).AddSlots(-10).Build()
	if model.Slots() != 0 {
		t.Fatalf("Slots should clamp at 0, was %d", model.Slots())
	}
}

func TestAddLevelClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetLevel(3).AddLevel(-5).Build()
	if model.Level() != 0 {
		t.Fatalf("Level should clamp at 0, was %d", model.Level())
	}
}

func TestAddLevelClampsAtMax(t *testing.T) {
	model := equipable.NewBuilder(1).SetLevel(250).AddLevel(10).Build()
	if model.Level() != 255 {
		t.Fatalf("Level should clamp at 255, was %d", model.Level())
	}
}

func TestAddExperienceClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetExperience(1000).AddExperience(-2000).Build()
	if model.Experience() != 0 {
		t.Fatalf("Experience should clamp at 0, was %d", model.Experience())
	}
}

func TestAddHammersAppliedClampsAtZero(t *testing.T) {
	model := equipable.NewBuilder(1).SetHammersApplied(2).AddHammersApplied(-5).Build()
	if model.HammersApplied() != 0 {
		t.Fatalf("HammersApplied should clamp at 0, was %d", model.HammersApplied())
	}
}

func TestAddMethodsPositiveDelta(t *testing.T) {
	model := equipable.NewBuilder(1).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetHp(100).
		SetMp(100).
		SetWeaponAttack(10).
		SetMagicAttack(10).
		SetWeaponDefense(10).
		SetMagicDefense(10).
		SetAccuracy(10).
		SetAvoidability(10).
		SetHands(10).
		SetSpeed(10).
		SetJump(10).
		SetSlots(5).
		SetLevel(5).
		SetExperience(100).
		SetHammersApplied(1).
		AddStrength(5).
		AddDexterity(5).
		AddIntelligence(5).
		AddLuck(5).
		AddHp(50).
		AddMp(50).
		AddWeaponAttack(5).
		AddMagicAttack(5).
		AddWeaponDefense(5).
		AddMagicDefense(5).
		AddAccuracy(5).
		AddAvoidability(5).
		AddHands(5).
		AddSpeed(5).
		AddJump(5).
		AddSlots(2).
		AddLevel(2).
		AddExperience(50).
		AddHammersApplied(1).
		Build()

	if model.Strength() != 15 {
		t.Fatalf("Strength should be 15, was %d", model.Strength())
	}
	if model.Dexterity() != 15 {
		t.Fatalf("Dexterity should be 15, was %d", model.Dexterity())
	}
	if model.Intelligence() != 15 {
		t.Fatalf("Intelligence should be 15, was %d", model.Intelligence())
	}
	if model.Luck() != 15 {
		t.Fatalf("Luck should be 15, was %d", model.Luck())
	}
	if model.HP() != 150 {
		t.Fatalf("HP should be 150, was %d", model.HP())
	}
	if model.MP() != 150 {
		t.Fatalf("MP should be 150, was %d", model.MP())
	}
	if model.WeaponAttack() != 15 {
		t.Fatalf("WeaponAttack should be 15, was %d", model.WeaponAttack())
	}
	if model.MagicAttack() != 15 {
		t.Fatalf("MagicAttack should be 15, was %d", model.MagicAttack())
	}
	if model.WeaponDefense() != 15 {
		t.Fatalf("WeaponDefense should be 15, was %d", model.WeaponDefense())
	}
	if model.MagicDefense() != 15 {
		t.Fatalf("MagicDefense should be 15, was %d", model.MagicDefense())
	}
	if model.Accuracy() != 15 {
		t.Fatalf("Accuracy should be 15, was %d", model.Accuracy())
	}
	if model.Avoidability() != 15 {
		t.Fatalf("Avoidability should be 15, was %d", model.Avoidability())
	}
	if model.Hands() != 15 {
		t.Fatalf("Hands should be 15, was %d", model.Hands())
	}
	if model.Speed() != 15 {
		t.Fatalf("Speed should be 15, was %d", model.Speed())
	}
	if model.Jump() != 15 {
		t.Fatalf("Jump should be 15, was %d", model.Jump())
	}
	if model.Slots() != 7 {
		t.Fatalf("Slots should be 7, was %d", model.Slots())
	}
	if model.Level() != 7 {
		t.Fatalf("Level should be 7, was %d", model.Level())
	}
	if model.Experience() != 150 {
		t.Fatalf("Experience should be 150, was %d", model.Experience())
	}
	if model.HammersApplied() != 2 {
		t.Fatalf("HammersApplied should be 2, was %d", model.HammersApplied())
	}
}
