package compartment_test

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/data/consumable"
	dcp "atlas-inventory/data/consumable/mock"
	"atlas-inventory/kafka/message"
	"context"
	"os"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	compartment.InitReservationRegistry(rc)
	compartment.InitLockRegistry(rc)
	os.Exit(m.Run())
}

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, asset.Migration, compartment.Migration)

	for _, migrator := range migrators {
		if err := migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
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

// TestCompactAndSort tests the behavior of the CompactAndSort function
// This test verifies that the CompactAndSort function correctly compacts and sorts assets by template ID
func TestCompactAndSort(t *testing.T) {
	// Create a character ID
	characterId := uint32(1)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		rm := consumable.RestModel{SlotMax: 100}
		m, err := consumable.Extract(rm)
		if err != nil {
			return consumable.Model{}, err
		}
		return m, nil
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	var err error
	_, err = cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	// Create assets with gaps in slots
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	// Create an asset with a higher template ID but in a higher slot
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2070000, 3, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 2: %v", err)
	}

	// Call CompactAndSort
	err = cp.CompactAndSort(mb)(uuid.New(), characterId, inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to compact and sort assets: %v", err)
	}

	// Verify that the assets were compacted and sorted
	c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to get compartment: %v", err)
	}

	// Verify that the assets are in the correct slots and sorted by template ID
	for _, a := range c.Assets() {
		if a.TemplateId() == 2070000 && a.Slot() != 1 {
			t.Fatalf("Asset 2070000 was not moved to slot 1")
		}
		if a.TemplateId() == 2120000 && a.Slot() != 2 {
			t.Fatalf("Asset 2120000 was not moved to slot 2")
		}
	}
}

// TestSort tests the behavior of the CompactAndSort function
func TestSort(t *testing.T) {
	// Create a character ID
	characterId := uint32(1)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		rm := consumable.RestModel{SlotMax: 100}
		m, err := consumable.Extract(rm)
		if err != nil {
			return consumable.Model{}, err
		}
		return m, nil
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	var err error
	_, err = cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	// Create two assets with the same template ID but in different slots
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	// Create an asset with a lower template ID but in a higher slot
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2070000, 5, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 3: %v", err)
	}

	// Call CompactAndSort
	err = cp.CompactAndSort(mb)(uuid.New(), characterId, inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to merge and sort assets: %v", err)
	}

	// Verify that the assets were merged, compacted, and sorted
	c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to get compartment: %v", err)
	}

	// Verify that the assets are in the correct slots and sorted by template ID
	for _, a := range c.Assets() {
		if a.TemplateId() == 2070000 && a.Slot() != 1 {
			t.Fatalf("Asset 2070000 was not moved to slot 1")
		}
		if a.TemplateId() == 2120000 && a.Slot() != 2 {
			t.Fatalf("Asset 2120000 was not moved to slot 2")
		}
	}
}

// TestMergeAndCompact tests the behavior of the MergeAndCompact function
// This test verifies that the MergeAndSort function correctly sorts assets by template ID
func TestMergeAndCompact(t *testing.T) {
	// Create a character ID
	characterId := uint32(1)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		rm := consumable.RestModel{SlotMax: 100}
		m, err := consumable.Extract(rm)
		if err != nil {
			return consumable.Model{}, err
		}
		return m, nil
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	var err error
	_, err = cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	err = cp.MergeAndCompact(mb)(uuid.New(), characterId, inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to merge and sort assets: %v", err)
	}

	c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to get compartment: %v", err)
	}
	for _, a := range c.Assets() {
		if a.TemplateId() == 2120000 && a.Slot() != 1 && a.Quantity() != 3 {
			t.Fatalf("Asset 2120000 was not merged to slot 1 correctly")
		}
	}
}

// TestMergeAndCompactOverflow tests the behavior of the MergeAndCompact function
// This test verifies that the MergeAndSort function correctly sorts assets by template ID
func TestMergeAndCompactOverflow(t *testing.T) {
	// Create a character ID
	characterId := uint32(1)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		rm := consumable.RestModel{SlotMax: 100}
		m, err := consumable.Extract(rm)
		if err != nil {
			return consumable.Model{}, err
		}
		return m, nil
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	var err error
	_, err = cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	err = cp.MergeAndCompact(mb)(uuid.New(), characterId, inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to merge and sort assets: %v", err)
	}

	c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to get compartment: %v", err)
	}
	for _, a := range c.Assets() {
		if a.TemplateId() == 2120000 && a.Slot() == 1 && a.Quantity() != 100 {
			t.Fatalf("Asset 2120000 was not merged to slot 1 correctly")
		}
		if a.TemplateId() == 2120000 && a.Slot() == 2 && a.Quantity() != 50 {
			t.Fatalf("Asset 2120000 was not merged to slot 2 correctly")
		}
	}
}

// TestMergeAndCompactGood tests the behavior of the MergeAndCompact function
// This test verifies that the MergeAndSort function correctly sorts assets by template ID
func TestMergeAndCompactGood(t *testing.T) {
	// Create a character ID
	characterId := uint32(1)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		rm := consumable.RestModel{SlotMax: 100}
		m, err := consumable.Extract(rm)
		if err != nil {
			return consumable.Model{}, err
		}
		return m, nil
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	var err error
	_, err = cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 100, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	err = cp.MergeAndCompact(mb)(uuid.New(), characterId, inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to merge and sort assets: %v", err)
	}

	c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to get compartment: %v", err)
	}
	for _, a := range c.Assets() {
		if a.TemplateId() == 2120000 && a.Slot() == 1 && a.Quantity() != 100 {
			t.Fatalf("Asset 2120000 was not merged to slot 1 correctly")
		}
		if a.TemplateId() == 2120000 && a.Slot() == 2 && a.Quantity() != 50 {
			t.Fatalf("Asset 2120000 was not merged to slot 2 correctly")
		}
	}
}
