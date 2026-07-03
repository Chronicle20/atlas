package compartment_test

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/data/consumable"
	dcp "atlas-inventory/data/consumable/mock"
	"atlas-inventory/kafka/message"
	assetMsg "atlas-inventory/kafka/message/asset"
	compartmentMsg "atlas-inventory/kafka/message/compartment"
	dropMsg "atlas-inventory/kafka/message/drop"
	pickupMsg "atlas-inventory/kafka/message/pickup"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	// Failure-path rejections/commands (Accept/Release/AttemptXPickUp) fire
	// via the DIRECT producer path (see D7 fix). Swap in a no-op writer so
	// those real-Kafka calls succeed instantly instead of retrying against
	// an unreachable broker for ~42s (see producertest package doc).
	producertest.InstallNoop()

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

func testDatabase(t *testing.T, l logrus.FieldLogger) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

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
	db := testDatabase(t, l)

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
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	// Create an asset with a higher template ID but in a higher slot
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2070000, 3, time.Time{}, 0, 0, 0, false)
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
	db := testDatabase(t, l)

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
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}

	// Create an asset with a lower template ID but in a higher slot
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2070000, 5, time.Time{}, 0, 0, 0, false)
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
	db := testDatabase(t, l)

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
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 1, time.Time{}, 0, 0, 0, false)
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
	db := testDatabase(t, l)

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
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0, false)
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

// TestConsumeRechargeablePreservesRow verifies that consuming the final unit of a
// rechargeable item (throwing stars, bullets) retains the row at qty=0 rather than
// deleting it — required so players can recharge the empty stack at an NPC shop.
func TestConsumeRechargeablePreservesRow(t *testing.T) {
	cases := []struct {
		name        string
		characterId uint32
		templateId  uint32
	}{
		{"throwing star", 101, 2070000},
		{"bullet", 102, 2330000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := testLogger()
			te := testTenant()
			ctx := tenant.WithContext(context.Background(), te)
			db := testDatabase(t, l)

			mb := message.NewBuffer()

			dcpi := &dcp.ProcessorImpl{}
			dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
				return consumable.Extract(consumable.RestModel{SlotMax: 100})
			}

			ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
			cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

			if _, err := cp.Create(mb)(uuid.New(), tc.characterId, inventory.TypeValueUse, 40); err != nil {
				t.Fatalf("Failed to create compartment: %v", err)
			}
			slot := int16(1)
			if err := cp.CreateAsset(mb)(uuid.New(), tc.characterId, inventory.TypeValueUse, tc.templateId, 1, time.Time{}, 0, 0, 0, false); err != nil {
				t.Fatalf("Failed to create asset: %v", err)
			}

			txId := uuid.New()
			if err := cp.RequestReserve(mb)(txId, tc.characterId, inventory.TypeValueUse, []compartment.ReservationRequest{{Slot: slot, ItemId: tc.templateId, Quantity: 1}}); err != nil {
				t.Fatalf("Failed to reserve: %v", err)
			}
			if err := cp.ConsumeAsset(mb)(txId, tc.characterId, inventory.TypeValueUse, slot); err != nil {
				t.Fatalf("Failed to consume: %v", err)
			}

			c, err := cp.GetByCharacterAndType(tc.characterId)(inventory.TypeValueUse)
			if err != nil {
				t.Fatalf("Failed to get compartment: %v", err)
			}
			found := false
			for _, a := range c.Assets() {
				if a.Slot() == slot {
					found = true
					if a.Quantity() != 0 {
						t.Fatalf("%s row should be qty=0 after consume-last, got %d", tc.name, a.Quantity())
					}
				}
			}
			if !found {
				t.Fatalf("%s row was deleted — expected to be retained at qty=0", tc.name)
			}
		})
	}
}

// TestConsumeNonRechargeableDeletes is a regression guard: the delete-at-zero behavior
// still applies for every non-rechargeable consumable — our change is gated on
// item.IsRechargeable.
func TestConsumeNonRechargeableDeletes(t *testing.T) {
	characterId := uint32(1)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		return consumable.Extract(consumable.RestModel{SlotMax: 100})
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40); err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}
	// 2000000 — classification 200, generic consumable (potion family), not rechargeable.
	if err := cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2000000, 1, time.Time{}, 0, 0, 0, false); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	txId := uuid.New()
	slot := int16(1)
	if err := cp.RequestReserve(mb)(txId, characterId, inventory.TypeValueUse, []compartment.ReservationRequest{{Slot: slot, ItemId: 2000000, Quantity: 1}}); err != nil {
		t.Fatalf("Failed to reserve: %v", err)
	}
	if err := cp.ConsumeAsset(mb)(txId, characterId, inventory.TypeValueUse, slot); err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}

	c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueUse)
	if err != nil {
		t.Fatalf("Failed to get compartment: %v", err)
	}
	for _, a := range c.Assets() {
		if a.Slot() == slot {
			t.Fatalf("non-rechargeable row should have been deleted after consume-last")
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
	db := testDatabase(t, l)

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
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 100, time.Time{}, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("Failed to create asset 1: %v", err)
	}
	err = cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2120000, 50, time.Time{}, 0, 0, 0, false)
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

func testFieldModel() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

// TestDropRechargeableWithZeroQuantity is a regression guard for the "cannot drop
// empty throwing-star stack" bug: when a rechargeable stack has been fully
// consumed (quantity=0, row retained for NPC recharge), the client still sends
// quantity=1 when the player drags it to the ground. The server must drop the
// whole asset regardless of remaining charges.
func TestDropRechargeableWithZeroQuantity(t *testing.T) {
	cases := []struct {
		name        string
		characterId uint32
		templateId  uint32
	}{
		{"throwing star", 201, 2070015},
		{"bullet", 202, 2330000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := testLogger()
			te := testTenant()
			ctx := tenant.WithContext(context.Background(), te)
			db := testDatabase(t, l)

			mb := message.NewBuffer()

			dcpi := &dcp.ProcessorImpl{}
			dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
				return consumable.Extract(consumable.RestModel{SlotMax: 100})
			}

			ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
			cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

			if _, err := cp.Create(mb)(uuid.New(), tc.characterId, inventory.TypeValueUse, 40); err != nil {
				t.Fatalf("Failed to create compartment: %v", err)
			}
			slot := int16(1)
			if err := cp.CreateAsset(mb)(uuid.New(), tc.characterId, inventory.TypeValueUse, tc.templateId, 1, time.Time{}, 0, 0, 0, false); err != nil {
				t.Fatalf("Failed to create asset: %v", err)
			}

			consumeTx := uuid.New()
			if err := cp.RequestReserve(mb)(consumeTx, tc.characterId, inventory.TypeValueUse, []compartment.ReservationRequest{{Slot: slot, ItemId: tc.templateId, Quantity: 1}}); err != nil {
				t.Fatalf("Failed to reserve: %v", err)
			}
			if err := cp.ConsumeAsset(mb)(consumeTx, tc.characterId, inventory.TypeValueUse, slot); err != nil {
				t.Fatalf("Failed to consume: %v", err)
			}

			c, err := cp.GetByCharacterAndType(tc.characterId)(inventory.TypeValueUse)
			if err != nil {
				t.Fatalf("Failed to get compartment: %v", err)
			}
			retainedAtZero := false
			for _, a := range c.Assets() {
				if a.Slot() == slot && a.Quantity() == 0 {
					retainedAtZero = true
				}
			}
			if !retainedAtZero {
				t.Fatalf("precondition failed: rechargeable row must be retained at qty=0 before drop")
			}

			if err := cp.Drop(mb)(uuid.New(), tc.characterId, inventory.TypeValueUse, testFieldModel(), 100, 200, slot, 1); err != nil {
				t.Fatalf("Drop failed for empty rechargeable stack: %v", err)
			}

			c, err = cp.GetByCharacterAndType(tc.characterId)(inventory.TypeValueUse)
			if err != nil {
				t.Fatalf("Failed to get compartment after drop: %v", err)
			}
			for _, a := range c.Assets() {
				if a.Slot() == slot {
					t.Fatalf("%s row should have been deleted after drop", tc.name)
				}
			}
		})
	}
}

// TestDropNonRechargeableInsufficientQuantity is a regression guard: the
// "cannot drop more than what is owned" rule still applies for non-rechargeable
// items — the rechargeable bypass must not leak into generic consumables.
func TestDropNonRechargeableInsufficientQuantity(t *testing.T) {
	characterId := uint32(203)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		return consumable.Extract(consumable.RestModel{SlotMax: 100})
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 40); err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}
	if err := cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2000000, 1, time.Time{}, 0, 0, 0, false); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.Drop(mb)(uuid.New(), characterId, inventory.TypeValueUse, testFieldModel(), 0, 0, int16(1), 5); err == nil {
		t.Fatalf("expected drop to fail when quantity exceeds owned")
	}
}

// TestAttemptItemPickUpInventoryFull guards two related behaviors in the
// drop pickup failure path:
//
//  1. The inner CreateAsset's CREATION_FAILED rejection (with the dedicated
//     INVENTORY_FULL error code, not the generic UNKNOWN_ERROR — so
//     atlas-channel can render the right client status message) must NOT be
//     enqueued on the caller-supplied (outbox-bound in production) mb. Per
//     D7, a rejection reflecting no committed state change (the inner tx
//     rolled back) must not ride into the outbox. It now fires via
//     producer.ProviderImpl on a fresh, throwaway buffer alongside the
//     CANCEL_RESERVATION command below (TestMain installs
//     producertest.InstallNoop so that direct send succeeds instantly
//     instead of retrying against an unreachable broker); this test can only
//     observe its ABSENCE from mb, not its delivery.
//  2. The outer AttemptItemPickUp must NOT put the drop-side
//     CANCEL_RESERVATION command on the caller-supplied mb. Per D7, a
//     COMMAND to another service (atlas-drop) reflecting a failed/rolled-back
//     pickup must go on the DIRECT producer path, not ride into the outbox
//     alongside a state change that never happened. It now fires via
//     producer.ProviderImpl on its own throwaway buffer; this test can only
//     observe its ABSENCE from mb, not its delivery.
func TestAttemptItemPickUpInventoryFull(t *testing.T) {
	characterId := uint32(304)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		return consumable.Extract(consumable.RestModel{SlotMax: 100})
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	// Capacity-1 USE compartment that is already full with a different
	// templateId, forcing the new pickup to need a fresh slot.
	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 1); err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}
	if err := cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2000000, 1, time.Time{}, 0, 0, 0, false); err != nil {
		t.Fatalf("Failed to seed compartment with existing item: %v", err)
	}

	// Reset the buffer so we only inspect events from the failing pickup.
	mb = message.NewBuffer()

	// 2070000 (subi throwing-stars classification ≠ 2000000), guaranteed not to
	// merge into the existing stack so the path falls through to NextFreeSlot.
	if err := cp.AttemptItemPickUp(mb)(uuid.New(), testFieldModel(), characterId, uint32(42), uint32(2070000), uint32(1)); err != nil {
		t.Fatalf("AttemptItemPickUp returned unexpected error: %v", err)
	}

	events := mb.GetAll()

	// D7: CREATION_FAILED is a rejection reflecting no committed state
	// change (the inner tx rolled back); it must not be enqueued in the
	// outbox-bound buffer. It now fires via the direct producer path.
	statusMsgs := events[compartmentMsg.EnvEventTopicStatus]
	for _, msg := range statusMsgs {
		var ev compartmentMsg.StatusEvent[compartmentMsg.CreationFailedStatusEventBody]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if ev.Type == compartmentMsg.StatusEventTypeCreationFailed {
			t.Fatalf("CREATION_FAILED must not be enqueued in the outbox-bound buffer; it must go via the direct producer path (D7)")
		}
	}

	// D7: CANCEL_RESERVATION is a cross-service COMMAND fired on a failed
	// (rolled-back) pickup attempt; it must be routed via the DIRECT
	// producer path, not enqueued in the outbox-bound mb.
	dropCmds := events[dropMsg.EnvCommandTopic]
	for _, msg := range dropCmds {
		var cmd dropMsg.Command[dropMsg.CancelReservationCommandBody]
		if err := json.Unmarshal(msg.Value, &cmd); err != nil {
			continue
		}
		if cmd.Type == dropMsg.CommandTypeCancelReservation {
			t.Fatalf("CANCEL_RESERVATION must not be enqueued in the outbox-bound buffer; it must go via the direct producer path (D7)")
		}
	}
}

// TestAttemptItemPickUpSuccess guards the merge-on-success side of the D7
// scratch-buffer fix: when the inner tx commits, the inner CreateAsset's
// ASSET CREATED event (buffered on a scratch innerMb, per the fix) must
// still land in the caller-supplied (outbox-bound in production) mb
// alongside the drop-side REQUEST_PICK_UP follow-on command. Regression
// guard for the AttemptItemPickUpInventoryFull fix, which introduced the
// scratch buffer and must not accidentally drop events on the success path.
func TestAttemptItemPickUpSuccess(t *testing.T) {
	characterId := uint32(306)
	templateId := uint32(2070000) // subi throwing-stars, USE inventory

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		return consumable.Extract(consumable.RestModel{SlotMax: 100})
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 24); err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	// Reset the buffer so we only inspect events from the pickup itself.
	mb = message.NewBuffer()

	if err := cp.AttemptItemPickUp(mb)(uuid.New(), testFieldModel(), characterId, uint32(44), templateId, uint32(1)); err != nil {
		t.Fatalf("AttemptItemPickUp returned unexpected error: %v", err)
	}

	events := mb.GetAll()

	assetStatusMsgs := events[assetMsg.EnvEventTopicStatus]
	var sawAssetCreated bool
	for _, msg := range assetStatusMsgs {
		var ev assetMsg.StatusEvent[assetMsg.CreatedStatusEventBody]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if ev.Type == assetMsg.StatusEventTypeCreated {
			sawAssetCreated = true
		}
	}
	if !sawAssetCreated {
		t.Fatalf("expected the inner CreateAsset's ASSET CREATED event to land in the outbox-bound mb on success, got %d events on topic", len(assetStatusMsgs))
	}

	dropCmds := events[dropMsg.EnvCommandTopic]
	var sawRequestPickUp bool
	for _, msg := range dropCmds {
		var cmd dropMsg.Command[dropMsg.RequestPickUpCommandBody]
		if err := json.Unmarshal(msg.Value, &cmd); err != nil {
			continue
		}
		if cmd.Type == dropMsg.CommandTypeRequestPickUp {
			sawRequestPickUp = true
		}
	}
	if !sawRequestPickUp {
		t.Fatalf("expected a REQUEST_PICK_UP drop command in the outbox-bound mb on success, got %d events on topic", len(dropCmds))
	}
}

// TestAttemptItemPickUpConsumeOnPickup verifies that USE-type items flagged
// with consumeOnPickup never enter the inventory. Instead the processor must
// emit an ITEM_CONSUMED_ON_PICKUP command and release the drop reservation
// via RequestPickUp (not CancelReservation).
func TestAttemptItemPickUpConsumeOnPickup(t *testing.T) {
	characterId := uint32(305)
	templateId := uint32(2022000) // monster-book reward in the USE classification

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		return consumable.Extract(consumable.RestModel{SlotMax: 100, ConsumeOnPickup: true})
	}

	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	// Create the USE compartment so the lookup that would normally happen
	// in AttemptItemPickUp does not blow up if the branch ever falls through.
	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 24); err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	// Reset the buffer so we only inspect events from the pickup itself.
	mb = message.NewBuffer()

	transactionId := uuid.New()
	if err := cp.AttemptItemPickUp(mb)(transactionId, testFieldModel(), characterId, uint32(43), templateId, uint32(1)); err != nil {
		t.Fatalf("AttemptItemPickUp returned unexpected error: %v", err)
	}

	events := mb.GetAll()

	// 1. ITEM_CONSUMED_ON_PICKUP command emitted.
	pickupCmds := events[pickupMsg.EnvCommandTopic]
	if len(pickupCmds) == 0 {
		t.Fatalf("expected an ITEM_CONSUMED_ON_PICKUP command, got none")
	}
	var cmd pickupMsg.Command
	if err := json.Unmarshal(pickupCmds[0].Value, &cmd); err != nil {
		t.Fatalf("failed to unmarshal pickup command: %v", err)
	}
	if cmd.Type != pickupMsg.CommandType {
		t.Fatalf("pickup command type = %q, want %q", cmd.Type, pickupMsg.CommandType)
	}
	if cmd.CharacterId != characterId {
		t.Fatalf("pickup command characterId = %d, want %d", cmd.CharacterId, characterId)
	}
	if cmd.ItemId != templateId {
		t.Fatalf("pickup command itemId = %d, want %d", cmd.ItemId, templateId)
	}
	if cmd.TransactionId != transactionId {
		t.Fatalf("pickup command transactionId mismatch")
	}
	if cmd.TenantId != te.Id() {
		t.Fatalf("pickup command tenantId = %s, want %s", cmd.TenantId, te.Id())
	}

	// 2. Drop reservation released via RequestPickUp (not CancelReservation).
	dropCmds := events[dropMsg.EnvCommandTopic]
	var sawRequestPickUp, sawCancel bool
	for _, msg := range dropCmds {
		var generic dropMsg.Command[json.RawMessage]
		if err := json.Unmarshal(msg.Value, &generic); err != nil {
			continue
		}
		if generic.Type == dropMsg.CommandTypeRequestPickUp {
			sawRequestPickUp = true
		}
		if generic.Type == dropMsg.CommandTypeCancelReservation {
			sawCancel = true
		}
	}
	if !sawRequestPickUp {
		t.Fatalf("expected a REQUEST_PICKUP drop command, got %d drop commands", len(dropCmds))
	}
	if sawCancel {
		t.Fatalf("did not expect a CANCEL_RESERVATION drop command on the consume-on-pickup branch")
	}

	// 3. No inventory CREATED status for the consumed-on-pickup item.
	statusMsgs := events[compartmentMsg.EnvEventTopicStatus]
	for _, msg := range statusMsgs {
		var ev compartmentMsg.StatusEvent[json.RawMessage]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		// The Create() call above legitimately produced one CREATED event for the
		// compartment itself; that buffer was already drained. Any CREATED event
		// in this buffer would indicate a stray inventory mutation.
		if ev.Type == compartmentMsg.StatusEventTypeCreated {
			t.Fatalf("unexpected CREATED status event on consume-on-pickup path")
		}
	}
}

// TestAcceptCommandFailedRoutesDirect verifies that Accept's rejection event
// (ACCEPT_COMMAND_FAILED) does not land in the caller-supplied buffer when
// the inner tx fails/rolls back. Per D7 a rejection reflecting no committed
// state change must fire via the direct producer path, not ride into the
// outbox alongside a state change that never happened.
func TestAcceptCommandFailedRoutesDirect(t *testing.T) {
	characterId := uint32(306)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	cp := compartment.NewProcessor(l, ctx, db)

	// No compartment exists for this character/type, so the inner
	// GetByCharacterAndType lookup fails and Accept must reject.
	m := asset.NewBuilder(uuid.New(), 2000000).SetSlot(1).SetQuantity(1).Build()

	mb := message.NewBuffer()
	if err := cp.Accept(mb)(uuid.New(), characterId, inventory.TypeValueUse, m); err != nil {
		t.Fatalf("Accept returned unexpected error: %v", err)
	}

	events := mb.GetAll()
	statusMsgs := events[compartmentMsg.EnvEventTopicStatus]
	for _, msg := range statusMsgs {
		var ev compartmentMsg.StatusEvent[compartmentMsg.ErrorEventBody]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if ev.Type == compartmentMsg.StatusEventTypeError && ev.Body.ErrorCode == compartmentMsg.AcceptCommandFailed {
			t.Fatalf("ACCEPT_COMMAND_FAILED must not be enqueued in the outbox-bound buffer; it must go via the direct producer path (D7)")
		}
	}
}

// TestReleaseCommandFailedRoutesDirect verifies the same for Release's
// RELEASE_COMMAND_FAILED rejection.
func TestReleaseCommandFailedRoutesDirect(t *testing.T) {
	characterId := uint32(307)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	cp := compartment.NewProcessor(l, ctx, db)

	// No compartment exists for this character/type, so the inner
	// GetByCharacterAndType lookup fails and Release must reject.
	mb := message.NewBuffer()
	if err := cp.Release(mb)(uuid.New(), characterId, inventory.TypeValueUse, 1, 1); err != nil {
		t.Fatalf("Release returned unexpected error: %v", err)
	}

	events := mb.GetAll()
	statusMsgs := events[compartmentMsg.EnvEventTopicStatus]
	for _, msg := range statusMsgs {
		var ev compartmentMsg.StatusEvent[compartmentMsg.ErrorEventBody]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if ev.Type == compartmentMsg.StatusEventTypeError && ev.Body.ErrorCode == compartmentMsg.ReleaseCommandFailed {
			t.Fatalf("RELEASE_COMMAND_FAILED must not be enqueued in the outbox-bound buffer; it must go via the direct producer path (D7)")
		}
	}
}
