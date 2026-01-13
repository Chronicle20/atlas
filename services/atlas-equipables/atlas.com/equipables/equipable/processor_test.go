package equipable_test

import (
	"atlas-equipables/equipable"
	"atlas-equipables/kafka/message"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := equipable.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestCreateWithExplicitStats(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302000).
		SetStrength(10).
		SetDexterity(5).
		SetIntelligence(3).
		SetLuck(2).
		SetHp(100).
		SetMp(50).
		SetWeaponAttack(15).
		SetMagicAttack(8).
		SetWeaponDefense(20).
		SetMagicDefense(10).
		SetAccuracy(5).
		SetAvoidability(3).
		SetHands(2).
		SetSpeed(5).
		SetJump(3).
		SetSlots(7).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	if created.Id() == 0 {
		t.Fatal("Created equipable should have non-zero ID")
	}
	if created.ItemId() != 1302000 {
		t.Fatalf("ItemId should be 1302000, was %d", created.ItemId())
	}
	if created.Strength() != 10 {
		t.Fatalf("Strength should be 10, was %d", created.Strength())
	}
	if created.Dexterity() != 5 {
		t.Fatalf("Dexterity should be 5, was %d", created.Dexterity())
	}
	if created.WeaponAttack() != 15 {
		t.Fatalf("WeaponAttack should be 15, was %d", created.WeaponAttack())
	}
	if created.Slots() != 7 {
		t.Fatalf("Slots should be 7, was %d", created.Slots())
	}
}

func TestGetByIdSunny(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302001).
		SetStrength(20).
		SetSlots(5).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	retrieved, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve equipable with ID %d: %v", created.Id(), err)
	}

	if retrieved.Id() != created.Id() {
		t.Fatalf("Retrieved ID should be %d, was %d", created.Id(), retrieved.Id())
	}
	if retrieved.ItemId() != 1302001 {
		t.Fatalf("ItemId should be 1302001, was %d", retrieved.ItemId())
	}
	if retrieved.Strength() != 20 {
		t.Fatalf("Strength should be 20, was %d", retrieved.Strength())
	}
}

func TestGetByIdNotFound(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	p := equipable.NewProcessor(testLogger(), tctx, db)
	_, err := p.GetById(99999)
	if err == nil {
		t.Fatal("Expected error when retrieving non-existent equipable, but got none")
	}
}

func TestUpdateSunny(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302002).
		SetStrength(10).
		SetDexterity(5).
		SetSlots(7).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	updateInput := equipable.Clone(created).
		SetStrength(25).
		SetDexterity(15).
		Build()

	updated, err := p.Update(message.NewBuffer())(updateInput)
	if err != nil {
		t.Fatalf("Failed to update equipable: %v", err)
	}

	if updated.Strength() != 25 {
		t.Fatalf("Strength should be 25, was %d", updated.Strength())
	}
	if updated.Dexterity() != 15 {
		t.Fatalf("Dexterity should be 15, was %d", updated.Dexterity())
	}
	if updated.Slots() != 7 {
		t.Fatalf("Slots should be preserved as 7, was %d", updated.Slots())
	}
}

func TestUpdatePreservesUnchangedFields(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Note: Create only saves stat fields (strength, dex, etc.), not meta fields like OwnerName
	input := equipable.NewBuilder(0).
		SetItemId(1302003).
		SetStrength(50).
		SetDexterity(40).
		SetIntelligence(30).
		SetLuck(20).
		SetHp(500).
		SetMp(300).
		SetWeaponAttack(100).
		SetSlots(10).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	updateInput := equipable.Clone(created).
		SetStrength(60).
		Build()

	updated, err := p.Update(message.NewBuffer())(updateInput)
	if err != nil {
		t.Fatalf("Failed to update equipable: %v", err)
	}

	if updated.Strength() != 60 {
		t.Fatalf("Strength should be updated to 60, was %d", updated.Strength())
	}
	if updated.Dexterity() != 40 {
		t.Fatalf("Dexterity should be preserved as 40, was %d", updated.Dexterity())
	}
	if updated.Intelligence() != 30 {
		t.Fatalf("Intelligence should be preserved as 30, was %d", updated.Intelligence())
	}
	if updated.Luck() != 20 {
		t.Fatalf("Luck should be preserved as 20, was %d", updated.Luck())
	}
	if updated.HP() != 500 {
		t.Fatalf("HP should be preserved as 500, was %d", updated.HP())
	}
	if updated.WeaponAttack() != 100 {
		t.Fatalf("WeaponAttack should be preserved as 100, was %d", updated.WeaponAttack())
	}
	if updated.Slots() != 10 {
		t.Fatalf("Slots should be preserved as 10, was %d", updated.Slots())
	}
}

func TestUpdateNonExistentEquipable(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(99999).
		SetItemId(1302004).
		SetStrength(10).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	_, err := p.Update(message.NewBuffer())(input)
	if err == nil {
		t.Fatal("Expected error when updating non-existent equipable, but got none")
	}
}

func TestDeleteByIdSunny(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302005).
		SetStrength(10).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	err = p.DeleteById(message.NewBuffer())(created.Id())
	if err != nil {
		t.Fatalf("Failed to delete equipable: %v", err)
	}

	_, err = p.GetById(created.Id())
	if err == nil {
		t.Fatal("Expected error when retrieving deleted equipable, but got none")
	}
}

func TestDeleteByIdNonExistent(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	p := equipable.NewProcessor(testLogger(), tctx, db)
	// Note: GORM's Delete does not error when the record doesn't exist
	// This is expected behavior - delete is idempotent
	err := p.DeleteById(message.NewBuffer())(99999)
	if err != nil {
		t.Fatalf("Delete should not error for non-existent record: %v", err)
	}
}

func TestCreateMultipleEquipables(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	p := equipable.NewProcessor(testLogger(), tctx, db)

	input1 := equipable.NewBuilder(0).
		SetItemId(1302010).
		SetStrength(10).
		Build()

	input2 := equipable.NewBuilder(0).
		SetItemId(1302011).
		SetStrength(20).
		Build()

	created1, err := p.Create(message.NewBuffer())(input1)
	if err != nil {
		t.Fatalf("Failed to create first equipable: %v", err)
	}

	created2, err := p.Create(message.NewBuffer())(input2)
	if err != nil {
		t.Fatalf("Failed to create second equipable: %v", err)
	}

	if created1.Id() == created2.Id() {
		t.Fatal("Two created equipables should have different IDs")
	}

	retrieved1, err := p.GetById(created1.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve first equipable: %v", err)
	}

	retrieved2, err := p.GetById(created2.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve second equipable: %v", err)
	}

	if retrieved1.ItemId() != 1302010 {
		t.Fatalf("First equipable ItemId should be 1302010, was %d", retrieved1.ItemId())
	}

	if retrieved2.ItemId() != 1302011 {
		t.Fatalf("Second equipable ItemId should be 1302011, was %d", retrieved2.ItemId())
	}
}

func TestUpdateMultipleFields(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302006).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetSlots(7).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	updateInput := equipable.Clone(created).
		SetStrength(50).
		SetDexterity(40).
		SetIntelligence(30).
		SetLuck(20).
		SetSlots(5).
		SetOwnerName("NewOwner").
		SetLocked(true).
		Build()

	updated, err := p.Update(message.NewBuffer())(updateInput)
	if err != nil {
		t.Fatalf("Failed to update equipable: %v", err)
	}

	if updated.Strength() != 50 {
		t.Fatalf("Strength should be 50, was %d", updated.Strength())
	}
	if updated.Dexterity() != 40 {
		t.Fatalf("Dexterity should be 40, was %d", updated.Dexterity())
	}
	if updated.Intelligence() != 30 {
		t.Fatalf("Intelligence should be 30, was %d", updated.Intelligence())
	}
	if updated.Luck() != 20 {
		t.Fatalf("Luck should be 20, was %d", updated.Luck())
	}
	if updated.Slots() != 5 {
		t.Fatalf("Slots should be 5, was %d", updated.Slots())
	}
	if updated.OwnerName() != "NewOwner" {
		t.Fatalf("OwnerName should be 'NewOwner', was '%s'", updated.OwnerName())
	}
	if !updated.Locked() {
		t.Fatal("Locked should be true")
	}
}

func TestTenantIsolation(t *testing.T) {
	tenant1 := testTenant()
	tenant2, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	tctx1 := tenant.WithContext(context.Background(), tenant1)
	tctx2 := tenant.WithContext(context.Background(), tenant2)

	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302007).
		SetStrength(10).
		Build()

	p1 := equipable.NewProcessor(testLogger(), tctx1, db)
	created, err := p1.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	p2 := equipable.NewProcessor(testLogger(), tctx2, db)
	_, err = p2.GetById(created.Id())
	if err == nil {
		t.Fatal("Expected error when retrieving equipable from different tenant, but got none")
	}
}

func TestBooleanFieldsUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302008).
		SetStrength(10).
		SetLocked(false).
		SetSpikes(false).
		SetKarmaUsed(false).
		SetCold(false).
		SetCanBeTraded(true).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	updateInput := equipable.Clone(created).
		SetLocked(true).
		SetSpikes(true).
		SetKarmaUsed(true).
		SetCold(true).
		SetCanBeTraded(false).
		Build()

	updated, err := p.Update(message.NewBuffer())(updateInput)
	if err != nil {
		t.Fatalf("Failed to update equipable: %v", err)
	}

	if !updated.Locked() {
		t.Fatal("Locked should be true")
	}
	if !updated.Spikes() {
		t.Fatal("Spikes should be true")
	}
	if !updated.KarmaUsed() {
		t.Fatal("KarmaUsed should be true")
	}
	if !updated.Cold() {
		t.Fatal("Cold should be true")
	}
	if updated.CanBeTraded() {
		t.Fatal("CanBeTraded should be false")
	}
}

func TestLevelTypeAndExperienceUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	input := equipable.NewBuilder(0).
		SetItemId(1302009).
		SetStrength(10).
		SetLevelType(0).
		SetLevel(1).
		SetExperience(0).
		SetHammersApplied(0).
		Build()

	p := equipable.NewProcessor(testLogger(), tctx, db)
	created, err := p.Create(message.NewBuffer())(input)
	if err != nil {
		t.Fatalf("Failed to create equipable: %v", err)
	}

	updateInput := equipable.Clone(created).
		SetLevelType(1).
		SetLevel(5).
		SetExperience(1000).
		SetHammersApplied(3).
		Build()

	updated, err := p.Update(message.NewBuffer())(updateInput)
	if err != nil {
		t.Fatalf("Failed to update equipable: %v", err)
	}

	if updated.LevelType() != 1 {
		t.Fatalf("LevelType should be 1, was %d", updated.LevelType())
	}
	if updated.Level() != 5 {
		t.Fatalf("Level should be 5, was %d", updated.Level())
	}
	if updated.Experience() != 1000 {
		t.Fatalf("Experience should be 1000, was %d", updated.Experience())
	}
	if updated.HammersApplied() != 3 {
		t.Fatalf("HammersApplied should be 3, was %d", updated.HammersApplied())
	}
}
