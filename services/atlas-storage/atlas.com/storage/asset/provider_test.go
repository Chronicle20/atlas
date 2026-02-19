package asset_test

import (
	"atlas-storage/asset"
	"atlas-storage/storage"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/world"
	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testContext() context.Context {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	l, _ := test.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, storage.Migration, asset.Migration)

	for _, migrator := range migrators {
		if err = migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func createTestStorage(t *testing.T, db *gorm.DB, tenantId uuid.UUID, worldId world.Id, accountId uint32) storage.Model {
	s, err := storage.Create(testLogger(), db, tenantId)(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}
	return s
}

func TestAsset_Create(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	m := asset.NewBuilder(s.Id(), 1302000).
		SetSlot(1).
		SetExpiration(time.Now().Add(time.Hour * 24 * 365)).
		SetStrength(10).
		SetDexterity(5).
		Build()

	a, err := asset.Create(testLogger(), db, te.Id())(m)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if a.Id() == 0 {
		t.Fatalf("Asset ID should be generated")
	}
	if a.StorageId() != s.Id() {
		t.Fatalf("StorageId mismatch. Expected %s, got %s", s.Id(), a.StorageId())
	}
	if a.Slot() != 1 {
		t.Fatalf("Slot mismatch. Expected 1, got %d", a.Slot())
	}
	if a.TemplateId() != 1302000 {
		t.Fatalf("TemplateId mismatch. Expected 1302000, got %d", a.TemplateId())
	}
	if a.InventoryType() != inventory.TypeValueEquip {
		t.Fatalf("InventoryType mismatch. Expected %d, got %d", inventory.TypeValueEquip, a.InventoryType())
	}
}

func TestAsset_GetById(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	m := asset.NewBuilder(s.Id(), 1302000).
		SetSlot(1).
		SetExpiration(time.Now().Add(time.Hour * 24 * 365)).
		Build()

	created, err := asset.Create(testLogger(), db, te.Id())(m)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	retrieved, err := asset.GetById(db.WithContext(ctx))(created.Id())
	if err != nil {
		t.Fatalf("Failed to get asset by ID: %v", err)
	}

	if retrieved.Id() != created.Id() {
		t.Fatalf("ID mismatch. Expected %d, got %d", created.Id(), retrieved.Id())
	}
	if retrieved.TemplateId() != created.TemplateId() {
		t.Fatalf("TemplateId mismatch. Expected %d, got %d", created.TemplateId(), retrieved.TemplateId())
	}
}

func TestAsset_GetByStorageId(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	// Create multiple assets
	for i := 0; i < 3; i++ {
		m := asset.NewBuilder(s.Id(), uint32(1300000+i)).
			SetSlot(int16(i)).
			SetExpiration(time.Now().Add(time.Hour * 24 * 365)).
			Build()

		_, err := asset.Create(testLogger(), db, te.Id())(m)
		if err != nil {
			t.Fatalf("Failed to create asset %d: %v", i, err)
		}
	}

	assets, err := asset.GetByStorageId(db.WithContext(ctx))(s.Id())
	if err != nil {
		t.Fatalf("Failed to get assets by storage ID: %v", err)
	}

	if len(assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(assets))
	}
}

func TestAsset_Delete(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	m := asset.NewBuilder(s.Id(), 1302000).
		SetSlot(1).
		SetExpiration(time.Now().Add(time.Hour * 24 * 365)).
		Build()

	a, err := asset.Create(testLogger(), db, te.Id())(m)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	err = asset.Delete(testLogger(), db.WithContext(ctx))(a.Id())
	if err != nil {
		t.Fatalf("Failed to delete asset: %v", err)
	}

	_, err = asset.GetById(db.WithContext(ctx))(a.Id())
	if err == nil {
		t.Fatalf("Asset should have been deleted")
	}
}

func TestAsset_DeleteByStorageId(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	// Create multiple assets
	for i := 1; i <= 3; i++ {
		m := asset.NewBuilder(s.Id(), uint32(1300000+i)).
			SetSlot(int16(i)).
			SetExpiration(time.Now().Add(time.Hour * 24 * 365)).
			Build()

		_, err := asset.Create(testLogger(), db, te.Id())(m)
		if err != nil {
			t.Fatalf("Failed to create asset %d: %v", i, err)
		}
	}

	err := asset.DeleteByStorageId(testLogger(), db.WithContext(ctx))(s.Id())
	if err != nil {
		t.Fatalf("Failed to delete assets by storage ID: %v", err)
	}

	assets, err := asset.GetByStorageId(db.WithContext(ctx))(s.Id())
	if err != nil {
		t.Fatalf("Failed to get assets: %v", err)
	}

	if len(assets) != 0 {
		t.Fatalf("Expected 0 assets after deletion, got %d", len(assets))
	}
}

func TestAsset_UpdateSlot(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	m := asset.NewBuilder(s.Id(), 1302000).
		SetSlot(1).
		SetExpiration(time.Now().Add(time.Hour * 24 * 365)).
		Build()

	a, err := asset.Create(testLogger(), db, te.Id())(m)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	newSlot := int16(5)
	err = asset.UpdateSlot(testLogger(), db.WithContext(ctx))(a.Id(), newSlot)
	if err != nil {
		t.Fatalf("Failed to update slot: %v", err)
	}

	updated, err := asset.GetById(db.WithContext(ctx))(a.Id())
	if err != nil {
		t.Fatalf("Failed to get asset: %v", err)
	}

	if updated.Slot() != newSlot {
		t.Fatalf("Slot mismatch. Expected %d, got %d", newSlot, updated.Slot())
	}
}

func TestAsset_IsStackable(t *testing.T) {
	tests := []struct {
		name        string
		templateId  uint32
		isStackable bool
	}{
		{"equip", 1302000, false},       // equip (1xxx)
		{"consumable", 2000000, true},   // use (2xxx)
		{"setup", 3000000, true},        // setup (3xxx)
		{"etc", 4000000, true},          // etc (4xxx)
		{"cash", 5000000, false},        // cash (5xxx)
	}

	testStorageId := uuid.New()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := asset.NewBuilder(testStorageId, tc.templateId).Build()

			if a.IsStackable() != tc.isStackable {
				t.Fatalf("IsStackable() for %s should be %v", tc.name, tc.isStackable)
			}
		})
	}
}

func TestAsset_TypeChecks(t *testing.T) {
	tests := []struct {
		name          string
		templateId    uint32
		petId         uint32
		cashId        int64
		checkMethod   string
		expectedValue bool
	}{
		{"equip", 1302000, 0, 0, "IsEquipment", true},
		{"consumable", 2000000, 0, 0, "IsConsumable", true},
		{"setup", 3000000, 0, 0, "IsSetup", true},
		{"etc", 4000000, 0, 0, "IsEtc", true},
		{"cash", 5000000, 0, 0, "IsCash", true},
		{"pet", 5000000, 100, 0, "IsPet", true},
		{"cashEquip", 1302000, 0, 12345, "IsCashEquipment", true},
	}

	testStorageId := uuid.New()
	for _, tc := range tests {
		t.Run(tc.checkMethod, func(t *testing.T) {
			b := asset.NewBuilder(testStorageId, tc.templateId)
			if tc.petId > 0 {
				b.SetPetId(tc.petId)
			}
			if tc.cashId != 0 {
				b.SetCashId(tc.cashId)
			}
			a := b.Build()

			var result bool
			switch tc.checkMethod {
			case "IsEquipment":
				result = a.IsEquipment()
			case "IsCashEquipment":
				result = a.IsCashEquipment()
			case "IsConsumable":
				result = a.IsConsumable()
			case "IsSetup":
				result = a.IsSetup()
			case "IsEtc":
				result = a.IsEtc()
			case "IsCash":
				result = a.IsCash()
			case "IsPet":
				result = a.IsPet()
			}

			if result != tc.expectedValue {
				t.Fatalf("%s for %s should be %v", tc.checkMethod, tc.name, tc.expectedValue)
			}
		})
	}
}
