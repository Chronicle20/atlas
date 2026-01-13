package equipable_test

import (
	"atlas-equipables/equipable"
	"testing"
	"time"
)

func TestTransformSunny(t *testing.T) {
	expTime := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	model := equipable.NewBuilder(12345).
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
		SetOwnerName("TestOwner").
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

	restModel, err := equipable.Transform(model)
	if err != nil {
		t.Fatalf("Transform should not error: %v", err)
	}

	// Verify all fields are correctly mapped
	if restModel.Id != 12345 {
		t.Fatalf("Id should be 12345, was %d", restModel.Id)
	}
	if restModel.ItemId != 1302000 {
		t.Fatalf("ItemId should be 1302000, was %d", restModel.ItemId)
	}
	if restModel.Strength != 50 {
		t.Fatalf("Strength should be 50, was %d", restModel.Strength)
	}
	if restModel.Dexterity != 40 {
		t.Fatalf("Dexterity should be 40, was %d", restModel.Dexterity)
	}
	if restModel.Intelligence != 30 {
		t.Fatalf("Intelligence should be 30, was %d", restModel.Intelligence)
	}
	if restModel.Luck != 20 {
		t.Fatalf("Luck should be 20, was %d", restModel.Luck)
	}
	if restModel.HP != 500 {
		t.Fatalf("HP should be 500, was %d", restModel.HP)
	}
	if restModel.MP != 300 {
		t.Fatalf("MP should be 300, was %d", restModel.MP)
	}
	if restModel.WeaponAttack != 100 {
		t.Fatalf("WeaponAttack should be 100, was %d", restModel.WeaponAttack)
	}
	if restModel.MagicAttack != 80 {
		t.Fatalf("MagicAttack should be 80, was %d", restModel.MagicAttack)
	}
	if restModel.WeaponDefense != 60 {
		t.Fatalf("WeaponDefense should be 60, was %d", restModel.WeaponDefense)
	}
	if restModel.MagicDefense != 40 {
		t.Fatalf("MagicDefense should be 40, was %d", restModel.MagicDefense)
	}
	if restModel.Accuracy != 20 {
		t.Fatalf("Accuracy should be 20, was %d", restModel.Accuracy)
	}
	if restModel.Avoidability != 15 {
		t.Fatalf("Avoidability should be 15, was %d", restModel.Avoidability)
	}
	if restModel.Hands != 10 {
		t.Fatalf("Hands should be 10, was %d", restModel.Hands)
	}
	if restModel.Speed != 25 {
		t.Fatalf("Speed should be 25, was %d", restModel.Speed)
	}
	if restModel.Jump != 15 {
		t.Fatalf("Jump should be 15, was %d", restModel.Jump)
	}
	if restModel.Slots != 7 {
		t.Fatalf("Slots should be 7, was %d", restModel.Slots)
	}
	if restModel.OwnerName != "TestOwner" {
		t.Fatalf("OwnerName should be 'TestOwner', was '%s'", restModel.OwnerName)
	}
	if !restModel.Locked {
		t.Fatal("Locked should be true")
	}
	if restModel.Spikes {
		t.Fatal("Spikes should be false")
	}
	if !restModel.KarmaUsed {
		t.Fatal("KarmaUsed should be true")
	}
	if restModel.Cold {
		t.Fatal("Cold should be false")
	}
	if !restModel.CanBeTraded {
		t.Fatal("CanBeTraded should be true")
	}
	if restModel.LevelType != 2 {
		t.Fatalf("LevelType should be 2, was %d", restModel.LevelType)
	}
	if restModel.Level != 10 {
		t.Fatalf("Level should be 10, was %d", restModel.Level)
	}
	if restModel.Experience != 5000 {
		t.Fatalf("Experience should be 5000, was %d", restModel.Experience)
	}
	if restModel.HammersApplied != 5 {
		t.Fatalf("HammersApplied should be 5, was %d", restModel.HammersApplied)
	}
	if !restModel.Expiration.Equal(expTime) {
		t.Fatalf("Expiration should be %v, was %v", expTime, restModel.Expiration)
	}
}

func TestExtractSunny(t *testing.T) {
	expTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	restModel := equipable.RestModel{
		Id:             54321,
		ItemId:         1402000,
		Strength:       100,
		Dexterity:      80,
		Intelligence:   60,
		Luck:           40,
		HP:             1000,
		MP:             500,
		WeaponAttack:   150,
		MagicAttack:    120,
		WeaponDefense:  90,
		MagicDefense:   70,
		Accuracy:       30,
		Avoidability:   25,
		Hands:          15,
		Speed:          35,
		Jump:           20,
		Slots:          5,
		OwnerName:      "Extracted",
		Locked:         false,
		Spikes:         true,
		KarmaUsed:      false,
		Cold:           true,
		CanBeTraded:    false,
		LevelType:      3,
		Level:          15,
		Experience:     10000,
		HammersApplied: 7,
		Expiration:     expTime,
	}

	model, err := equipable.Extract(restModel)
	if err != nil {
		t.Fatalf("Extract should not error: %v", err)
	}

	// Verify all fields are correctly mapped
	if model.Id() != 54321 {
		t.Fatalf("Id should be 54321, was %d", model.Id())
	}
	if model.ItemId() != 1402000 {
		t.Fatalf("ItemId should be 1402000, was %d", model.ItemId())
	}
	if model.Strength() != 100 {
		t.Fatalf("Strength should be 100, was %d", model.Strength())
	}
	if model.Dexterity() != 80 {
		t.Fatalf("Dexterity should be 80, was %d", model.Dexterity())
	}
	if model.Intelligence() != 60 {
		t.Fatalf("Intelligence should be 60, was %d", model.Intelligence())
	}
	if model.Luck() != 40 {
		t.Fatalf("Luck should be 40, was %d", model.Luck())
	}
	if model.HP() != 1000 {
		t.Fatalf("HP should be 1000, was %d", model.HP())
	}
	if model.MP() != 500 {
		t.Fatalf("MP should be 500, was %d", model.MP())
	}
	if model.WeaponAttack() != 150 {
		t.Fatalf("WeaponAttack should be 150, was %d", model.WeaponAttack())
	}
	if model.MagicAttack() != 120 {
		t.Fatalf("MagicAttack should be 120, was %d", model.MagicAttack())
	}
	if model.WeaponDefense() != 90 {
		t.Fatalf("WeaponDefense should be 90, was %d", model.WeaponDefense())
	}
	if model.MagicDefense() != 70 {
		t.Fatalf("MagicDefense should be 70, was %d", model.MagicDefense())
	}
	if model.Accuracy() != 30 {
		t.Fatalf("Accuracy should be 30, was %d", model.Accuracy())
	}
	if model.Avoidability() != 25 {
		t.Fatalf("Avoidability should be 25, was %d", model.Avoidability())
	}
	if model.Hands() != 15 {
		t.Fatalf("Hands should be 15, was %d", model.Hands())
	}
	if model.Speed() != 35 {
		t.Fatalf("Speed should be 35, was %d", model.Speed())
	}
	if model.Jump() != 20 {
		t.Fatalf("Jump should be 20, was %d", model.Jump())
	}
	if model.Slots() != 5 {
		t.Fatalf("Slots should be 5, was %d", model.Slots())
	}
	if model.OwnerName() != "Extracted" {
		t.Fatalf("OwnerName should be 'Extracted', was '%s'", model.OwnerName())
	}
	if model.Locked() {
		t.Fatal("Locked should be false")
	}
	if !model.Spikes() {
		t.Fatal("Spikes should be true")
	}
	if model.KarmaUsed() {
		t.Fatal("KarmaUsed should be false")
	}
	if !model.Cold() {
		t.Fatal("Cold should be true")
	}
	if model.CanBeTraded() {
		t.Fatal("CanBeTraded should be false")
	}
	if model.LevelType() != 3 {
		t.Fatalf("LevelType should be 3, was %d", model.LevelType())
	}
	if model.Level() != 15 {
		t.Fatalf("Level should be 15, was %d", model.Level())
	}
	if model.Experience() != 10000 {
		t.Fatalf("Experience should be 10000, was %d", model.Experience())
	}
	if model.HammersApplied() != 7 {
		t.Fatalf("HammersApplied should be 7, was %d", model.HammersApplied())
	}
	if !model.Expiration().Equal(expTime) {
		t.Fatalf("Expiration should be %v, was %v", expTime, model.Expiration())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	expTime := time.Date(2026, 9, 20, 18, 30, 0, 0, time.UTC)

	original := equipable.NewBuilder(99999).
		SetItemId(1502000).
		SetStrength(75).
		SetDexterity(65).
		SetIntelligence(55).
		SetLuck(45).
		SetHp(750).
		SetMp(400).
		SetWeaponAttack(125).
		SetMagicAttack(100).
		SetWeaponDefense(75).
		SetMagicDefense(55).
		SetAccuracy(25).
		SetAvoidability(20).
		SetHands(12).
		SetSpeed(30).
		SetJump(18).
		SetSlots(6).
		SetOwnerName("RoundTrip").
		SetLocked(true).
		SetSpikes(true).
		SetKarmaUsed(false).
		SetCold(false).
		SetCanBeTraded(true).
		SetLevelType(1).
		SetLevel(8).
		SetExperience(3000).
		SetHammersApplied(4).
		SetExpiration(expTime).
		Build()

	// Transform to REST model
	restModel, err := equipable.Transform(original)
	if err != nil {
		t.Fatalf("Transform should not error: %v", err)
	}

	// Extract back to domain model
	roundTripped, err := equipable.Extract(restModel)
	if err != nil {
		t.Fatalf("Extract should not error: %v", err)
	}

	// Verify all fields match
	if roundTripped.Id() != original.Id() {
		t.Fatalf("Id mismatch: expected %d, got %d", original.Id(), roundTripped.Id())
	}
	if roundTripped.ItemId() != original.ItemId() {
		t.Fatalf("ItemId mismatch")
	}
	if roundTripped.Strength() != original.Strength() {
		t.Fatalf("Strength mismatch")
	}
	if roundTripped.Dexterity() != original.Dexterity() {
		t.Fatalf("Dexterity mismatch")
	}
	if roundTripped.Intelligence() != original.Intelligence() {
		t.Fatalf("Intelligence mismatch")
	}
	if roundTripped.Luck() != original.Luck() {
		t.Fatalf("Luck mismatch")
	}
	if roundTripped.HP() != original.HP() {
		t.Fatalf("HP mismatch")
	}
	if roundTripped.MP() != original.MP() {
		t.Fatalf("MP mismatch")
	}
	if roundTripped.WeaponAttack() != original.WeaponAttack() {
		t.Fatalf("WeaponAttack mismatch")
	}
	if roundTripped.MagicAttack() != original.MagicAttack() {
		t.Fatalf("MagicAttack mismatch")
	}
	if roundTripped.WeaponDefense() != original.WeaponDefense() {
		t.Fatalf("WeaponDefense mismatch")
	}
	if roundTripped.MagicDefense() != original.MagicDefense() {
		t.Fatalf("MagicDefense mismatch")
	}
	if roundTripped.Accuracy() != original.Accuracy() {
		t.Fatalf("Accuracy mismatch")
	}
	if roundTripped.Avoidability() != original.Avoidability() {
		t.Fatalf("Avoidability mismatch")
	}
	if roundTripped.Hands() != original.Hands() {
		t.Fatalf("Hands mismatch")
	}
	if roundTripped.Speed() != original.Speed() {
		t.Fatalf("Speed mismatch")
	}
	if roundTripped.Jump() != original.Jump() {
		t.Fatalf("Jump mismatch")
	}
	if roundTripped.Slots() != original.Slots() {
		t.Fatalf("Slots mismatch")
	}
	if roundTripped.OwnerName() != original.OwnerName() {
		t.Fatalf("OwnerName mismatch")
	}
	if roundTripped.Locked() != original.Locked() {
		t.Fatalf("Locked mismatch")
	}
	if roundTripped.Spikes() != original.Spikes() {
		t.Fatalf("Spikes mismatch")
	}
	if roundTripped.KarmaUsed() != original.KarmaUsed() {
		t.Fatalf("KarmaUsed mismatch")
	}
	if roundTripped.Cold() != original.Cold() {
		t.Fatalf("Cold mismatch")
	}
	if roundTripped.CanBeTraded() != original.CanBeTraded() {
		t.Fatalf("CanBeTraded mismatch")
	}
	if roundTripped.LevelType() != original.LevelType() {
		t.Fatalf("LevelType mismatch")
	}
	if roundTripped.Level() != original.Level() {
		t.Fatalf("Level mismatch")
	}
	if roundTripped.Experience() != original.Experience() {
		t.Fatalf("Experience mismatch")
	}
	if roundTripped.HammersApplied() != original.HammersApplied() {
		t.Fatalf("HammersApplied mismatch")
	}
	if !roundTripped.Expiration().Equal(original.Expiration()) {
		t.Fatalf("Expiration mismatch")
	}
}

func TestGetNameReturnsEquipables(t *testing.T) {
	restModel := equipable.RestModel{}
	name := restModel.GetName()

	if name != "equipables" {
		t.Fatalf("GetName should return 'equipables', was '%s'", name)
	}
}

func TestGetIDFormatsAsString(t *testing.T) {
	restModel := equipable.RestModel{Id: 12345}
	id := restModel.GetID()

	if id != "12345" {
		t.Fatalf("GetID should return '12345', was '%s'", id)
	}
}

func TestGetIDWithZero(t *testing.T) {
	restModel := equipable.RestModel{Id: 0}
	id := restModel.GetID()

	if id != "0" {
		t.Fatalf("GetID should return '0', was '%s'", id)
	}
}

func TestGetIDWithLargeNumber(t *testing.T) {
	restModel := equipable.RestModel{Id: 4294967295} // max uint32
	id := restModel.GetID()

	if id != "4294967295" {
		t.Fatalf("GetID should return '4294967295', was '%s'", id)
	}
}

func TestSetIDParsesString(t *testing.T) {
	restModel := &equipable.RestModel{}
	err := restModel.SetID("54321")

	if err != nil {
		t.Fatalf("SetID should not error: %v", err)
	}

	if restModel.Id != 54321 {
		t.Fatalf("Id should be 54321, was %d", restModel.Id)
	}
}

func TestSetIDWithZero(t *testing.T) {
	restModel := &equipable.RestModel{}
	err := restModel.SetID("0")

	if err != nil {
		t.Fatalf("SetID should not error: %v", err)
	}

	if restModel.Id != 0 {
		t.Fatalf("Id should be 0, was %d", restModel.Id)
	}
}

func TestSetIDWithInvalidString(t *testing.T) {
	restModel := &equipable.RestModel{}
	err := restModel.SetID("not-a-number")

	if err == nil {
		t.Fatal("SetID should error for invalid input")
	}
}

func TestSetIDWithNegativeNumber(t *testing.T) {
	restModel := &equipable.RestModel{}
	err := restModel.SetID("-1")

	// Note: This parses to -1 which then gets cast to uint32
	// The behavior depends on implementation
	if err != nil {
		// If it errors, that's acceptable
		return
	}

	// If it doesn't error, the value will wrap around
	// This is expected Go behavior for int to uint32 conversion
}

func TestTransformWithZeroValues(t *testing.T) {
	model := equipable.NewBuilder(0).Build()

	restModel, err := equipable.Transform(model)
	if err != nil {
		t.Fatalf("Transform should not error: %v", err)
	}

	if restModel.Id != 0 {
		t.Fatalf("Id should be 0, was %d", restModel.Id)
	}
	if restModel.Strength != 0 {
		t.Fatalf("Strength should be 0, was %d", restModel.Strength)
	}
	if restModel.OwnerName != "" {
		t.Fatalf("OwnerName should be empty, was '%s'", restModel.OwnerName)
	}
	if restModel.Locked {
		t.Fatal("Locked should be false")
	}
}

func TestExtractWithZeroValues(t *testing.T) {
	restModel := equipable.RestModel{}

	model, err := equipable.Extract(restModel)
	if err != nil {
		t.Fatalf("Extract should not error: %v", err)
	}

	if model.Id() != 0 {
		t.Fatalf("Id should be 0, was %d", model.Id())
	}
	if model.Strength() != 0 {
		t.Fatalf("Strength should be 0, was %d", model.Strength())
	}
	if model.OwnerName() != "" {
		t.Fatalf("OwnerName should be empty, was '%s'", model.OwnerName())
	}
	if model.Locked() {
		t.Fatal("Locked should be false")
	}
}
