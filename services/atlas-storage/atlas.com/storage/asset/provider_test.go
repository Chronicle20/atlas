package asset_test

import (
	"atlas-storage/asset"
	"atlas-storage/storage"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
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

	a, err := asset.Create(testLogger(), db, te.Id())(
		s.Id(),
		1,
		1302000,
		time.Now().Add(time.Hour*24*365),
		100,
		asset.ReferenceTypeEquipable,
	)
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
	if a.ReferenceType() != asset.ReferenceTypeEquipable {
		t.Fatalf("ReferenceType mismatch. Expected %s, got %s", asset.ReferenceTypeEquipable, a.ReferenceType())
	}
}

func TestAsset_GetById(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	created, err := asset.Create(testLogger(), db, te.Id())(
		s.Id(),
		1,
		1302000,
		time.Now().Add(time.Hour*24*365),
		100,
		asset.ReferenceTypeEquipable,
	)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	retrieved, err := asset.GetById(testLogger(), db, te.Id())(created.Id())
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

	// Create multiple assets (0-indexed slots)
	for i := 0; i < 3; i++ {
		_, err := asset.Create(testLogger(), db, te.Id())(
			s.Id(),
			int16(i),
			uint32(1300000+i),
			time.Now().Add(time.Hour*24*365),
			uint32(i*100),
			asset.ReferenceTypeEquipable,
		)
		if err != nil {
			t.Fatalf("Failed to create asset %d: %v", i, err)
		}
	}

	assets, err := asset.GetByStorageId(testLogger(), db, te.Id())(s.Id())
	if err != nil {
		t.Fatalf("Failed to get assets by storage ID: %v", err)
	}

	if len(assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(assets))
	}

	// Verify assets are ordered by slot (0-indexed)
	for i, a := range assets {
		expectedSlot := int16(i)
		if a.Slot() != expectedSlot {
			t.Fatalf("Asset %d slot mismatch. Expected %d, got %d", i, expectedSlot, a.Slot())
		}
	}
}

func TestAsset_Delete(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	te := tenant.MustFromContext(ctx)

	s := createTestStorage(t, db, te.Id(), 0, 12345)

	a, err := asset.Create(testLogger(), db, te.Id())(
		s.Id(),
		1,
		1302000,
		time.Now().Add(time.Hour*24*365),
		100,
		asset.ReferenceTypeEquipable,
	)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	err = asset.Delete(testLogger(), db, te.Id())(a.Id())
	if err != nil {
		t.Fatalf("Failed to delete asset: %v", err)
	}

	_, err = asset.GetById(testLogger(), db, te.Id())(a.Id())
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
		_, err := asset.Create(testLogger(), db, te.Id())(
			s.Id(),
			int16(i),
			uint32(1300000+i),
			time.Now().Add(time.Hour*24*365),
			uint32(i*100),
			asset.ReferenceTypeEquipable,
		)
		if err != nil {
			t.Fatalf("Failed to create asset %d: %v", i, err)
		}
	}

	err := asset.DeleteByStorageId(testLogger(), db, te.Id())(s.Id())
	if err != nil {
		t.Fatalf("Failed to delete assets by storage ID: %v", err)
	}

	assets, err := asset.GetByStorageId(testLogger(), db, te.Id())(s.Id())
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

	a, err := asset.Create(testLogger(), db, te.Id())(
		s.Id(),
		1,
		1302000,
		time.Now().Add(time.Hour*24*365),
		100,
		asset.ReferenceTypeEquipable,
	)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	newSlot := int16(5)
	err = asset.UpdateSlot(testLogger(), db, te.Id())(a.Id(), newSlot)
	if err != nil {
		t.Fatalf("Failed to update slot: %v", err)
	}

	updated, err := asset.GetById(testLogger(), db, te.Id())(a.Id())
	if err != nil {
		t.Fatalf("Failed to get asset: %v", err)
	}

	if updated.Slot() != newSlot {
		t.Fatalf("Slot mismatch. Expected %d, got %d", newSlot, updated.Slot())
	}
}

func TestAsset_IsStackable(t *testing.T) {
	tests := []struct {
		refType     asset.ReferenceType
		isStackable bool
	}{
		{asset.ReferenceTypeEquipable, false},
		{asset.ReferenceTypeCashEquipable, false},
		{asset.ReferenceTypeConsumable, true},
		{asset.ReferenceTypeSetup, true},
		{asset.ReferenceTypeEtc, true},
		{asset.ReferenceTypeCash, false},
		{asset.ReferenceTypePet, false},
	}

	testStorageId := uuid.New()
	for _, tc := range tests {
		t.Run(string(tc.refType), func(t *testing.T) {
			a := asset.NewModelBuilder[any]().
				SetStorageId(testStorageId).
				SetTemplateId(1000000).
				SetReferenceType(tc.refType).
				MustBuild()

			if a.IsStackable() != tc.isStackable {
				t.Fatalf("IsStackable() for %s should be %v", tc.refType, tc.isStackable)
			}
		})
	}
}

func TestAsset_TypeChecks(t *testing.T) {
	tests := []struct {
		refType       asset.ReferenceType
		checkMethod   string
		expectedValue bool
	}{
		{asset.ReferenceTypeEquipable, "IsEquipable", true},
		{asset.ReferenceTypeCashEquipable, "IsCashEquipable", true},
		{asset.ReferenceTypeConsumable, "IsConsumable", true},
		{asset.ReferenceTypeSetup, "IsSetup", true},
		{asset.ReferenceTypeEtc, "IsEtc", true},
		{asset.ReferenceTypeCash, "IsCash", true},
		{asset.ReferenceTypePet, "IsPet", true},
	}

	testStorageId := uuid.New()
	for _, tc := range tests {
		t.Run(tc.checkMethod, func(t *testing.T) {
			a := asset.NewModelBuilder[any]().
				SetStorageId(testStorageId).
				SetTemplateId(1000000).
				SetReferenceType(tc.refType).
				MustBuild()

			var result bool
			switch tc.checkMethod {
			case "IsEquipable":
				result = a.IsEquipable()
			case "IsCashEquipable":
				result = a.IsCashEquipable()
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
				t.Fatalf("%s for %s should be %v", tc.checkMethod, tc.refType, tc.expectedValue)
			}
		})
	}
}
