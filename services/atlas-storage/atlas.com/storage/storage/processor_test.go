package storage_test

import (
	"atlas-storage/asset"
	"atlas-storage/kafka/message"
	"atlas-storage/stackable"
	"atlas-storage/storage"
	"context"
	"testing"
	"time"

	assetConstants "github.com/Chronicle20/atlas-constants/asset"
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
	migrators = append(migrators, storage.Migration, asset.Migration, stackable.Migration)

	for _, migrator := range migrators {
		if err = migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func TestProcessor_GetOrCreateStorage_Create(t *testing.T) {
	p := storage.NewProcessor(testLogger(), testContext(), testDatabase(t))

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Test execution - should create new storage
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get or create storage: %v", err)
	}

	if s.Id() == uuid.Nil {
		t.Fatalf("Storage ID was not generated")
	}
	if s.WorldId() != worldId {
		t.Fatalf("WorldId mismatch. Expected %d, got %d", worldId, s.WorldId())
	}
	if s.AccountId() != accountId {
		t.Fatalf("AccountId mismatch. Expected %d, got %d", accountId, s.AccountId())
	}
	if s.Capacity() != 4 {
		t.Fatalf("Capacity should default to 4, got %d", s.Capacity())
	}
	if s.Mesos() != 0 {
		t.Fatalf("Mesos should default to 0, got %d", s.Mesos())
	}
}

func TestProcessor_GetOrCreateStorage_Get(t *testing.T) {
	p := storage.NewProcessor(testLogger(), testContext(), testDatabase(t))

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Create initial storage
	s1, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Should retrieve existing storage
	s2, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get storage: %v", err)
	}

	if s1.Id() != s2.Id() {
		t.Fatalf("Storage IDs should match. Expected %s, got %s", s1.Id(), s2.Id())
	}
}

func TestProcessor_Deposit_Equipable(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	body := message.DepositBody{
		Slot:          1,
		TemplateId:    1302000, // Example sword
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   100,
		ReferenceType: string(asset.ReferenceTypeEquipable),
	}

	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		t.Fatalf("Failed to deposit item: %v", err)
	}

	if assetId == 0 {
		t.Fatalf("Asset ID should not be 0")
	}

	// Verify asset was created
	te := tenant.MustFromContext(ctx)
	a, err := asset.GetById(testLogger(), db, te.Id())(assetId)
	if err != nil {
		t.Fatalf("Failed to get asset: %v", err)
	}

	// Note: Slot is computed dynamically via GetByStorageId, not stored
	if a.TemplateId() != body.TemplateId {
		t.Fatalf("TemplateId mismatch. Expected %d, got %d", body.TemplateId, a.TemplateId())
	}
	if a.ReferenceId() != body.ReferenceId {
		t.Fatalf("ReferenceId mismatch. Expected %d, got %d", body.ReferenceId, a.ReferenceId())
	}
}

func TestProcessor_Deposit_Stackable(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	body := message.DepositBody{
		Slot:          1,
		TemplateId:    2000000, // Example consumable
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   0,
		ReferenceType: string(asset.ReferenceTypeConsumable),
		ReferenceData: message.ReferenceData{
			Quantity: 50,
			OwnerId:  999,
			Flag:     0,
		},
	}

	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		t.Fatalf("Failed to deposit stackable item: %v", err)
	}

	// Verify stackable data was created
	s, err := stackable.GetByAssetId(testLogger(), db)(assetId)
	if err != nil {
		t.Fatalf("Failed to get stackable data: %v", err)
	}

	if s.Quantity() != body.ReferenceData.Quantity {
		t.Fatalf("Quantity mismatch. Expected %d, got %d", body.ReferenceData.Quantity, s.Quantity())
	}
	if s.OwnerId() != body.ReferenceData.OwnerId {
		t.Fatalf("OwnerId mismatch. Expected %d, got %d", body.ReferenceData.OwnerId, s.OwnerId())
	}
}

func TestProcessor_Withdraw_Full(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Deposit an item first
	body := message.DepositBody{
		Slot:          1,
		TemplateId:    1302000,
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   100,
		ReferenceType: string(asset.ReferenceTypeEquipable),
	}

	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		t.Fatalf("Failed to deposit item: %v", err)
	}

	// Withdraw the item
	withdrawBody := message.WithdrawBody{
		AssetId: assetConstants.Id(assetId),
	}

	err = p.Withdraw(worldId, accountId, withdrawBody)
	if err != nil {
		t.Fatalf("Failed to withdraw item: %v", err)
	}

	// Verify asset was deleted
	te := tenant.MustFromContext(ctx)
	_, err = asset.GetById(testLogger(), db, te.Id())(assetId)
	if err == nil {
		t.Fatalf("Asset should have been deleted")
	}
}

func TestProcessor_Withdraw_Partial(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Deposit a stackable item
	body := message.DepositBody{
		Slot:          1,
		TemplateId:    2000000,
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   0,
		ReferenceType: string(asset.ReferenceTypeConsumable),
		ReferenceData: message.ReferenceData{
			Quantity: 100,
			OwnerId:  0,
			Flag:     0,
		},
	}

	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		t.Fatalf("Failed to deposit item: %v", err)
	}

	// Partial withdrawal
	withdrawBody := message.WithdrawBody{
		AssetId:  assetConstants.Id(assetId),
		Quantity: 30,
	}

	err = p.Withdraw(worldId, accountId, withdrawBody)
	if err != nil {
		t.Fatalf("Failed to withdraw item: %v", err)
	}

	// Verify quantity was reduced
	s, err := stackable.GetByAssetId(testLogger(), db)(assetId)
	if err != nil {
		t.Fatalf("Stackable should still exist: %v", err)
	}

	expectedQuantity := uint32(70)
	if s.Quantity() != expectedQuantity {
		t.Fatalf("Quantity mismatch. Expected %d, got %d", expectedQuantity, s.Quantity())
	}
}

func TestProcessor_UpdateMesos_Set(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Create storage
	_, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Set mesos
	body := message.UpdateMesosBody{
		Mesos:     5000,
		Operation: "SET",
	}

	err = p.UpdateMesos(worldId, accountId, body)
	if err != nil {
		t.Fatalf("Failed to update mesos: %v", err)
	}

	// Verify
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get storage: %v", err)
	}

	if s.Mesos() != 5000 {
		t.Fatalf("Mesos mismatch. Expected 5000, got %d", s.Mesos())
	}
}

func TestProcessor_UpdateMesos_Add(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Create storage with initial mesos
	_, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Set initial mesos
	err = p.UpdateMesos(worldId, accountId, message.UpdateMesosBody{Mesos: 1000, Operation: "SET"})
	if err != nil {
		t.Fatalf("Failed to set mesos: %v", err)
	}

	// Add mesos
	err = p.UpdateMesos(worldId, accountId, message.UpdateMesosBody{Mesos: 500, Operation: "ADD"})
	if err != nil {
		t.Fatalf("Failed to add mesos: %v", err)
	}

	// Verify
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get storage: %v", err)
	}

	if s.Mesos() != 1500 {
		t.Fatalf("Mesos mismatch. Expected 1500, got %d", s.Mesos())
	}
}

func TestProcessor_UpdateMesos_Subtract(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Create storage with initial mesos
	_, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Set initial mesos
	err = p.UpdateMesos(worldId, accountId, message.UpdateMesosBody{Mesos: 1000, Operation: "SET"})
	if err != nil {
		t.Fatalf("Failed to set mesos: %v", err)
	}

	// Subtract mesos
	err = p.UpdateMesos(worldId, accountId, message.UpdateMesosBody{Mesos: 300, Operation: "SUBTRACT"})
	if err != nil {
		t.Fatalf("Failed to subtract mesos: %v", err)
	}

	// Verify
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get storage: %v", err)
	}

	if s.Mesos() != 700 {
		t.Fatalf("Mesos mismatch. Expected 700, got %d", s.Mesos())
	}
}

func TestProcessor_UpdateMesos_SubtractUnderflow(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Create storage with initial mesos
	_, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Set initial mesos
	err = p.UpdateMesos(worldId, accountId, message.UpdateMesosBody{Mesos: 100, Operation: "SET"})
	if err != nil {
		t.Fatalf("Failed to set mesos: %v", err)
	}

	// Subtract more than available (should clamp to 0)
	err = p.UpdateMesos(worldId, accountId, message.UpdateMesosBody{Mesos: 500, Operation: "SUBTRACT"})
	if err != nil {
		t.Fatalf("Failed to subtract mesos: %v", err)
	}

	// Verify
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get storage: %v", err)
	}

	if s.Mesos() != 0 {
		t.Fatalf("Mesos should be 0 on underflow, got %d", s.Mesos())
	}
}

func TestProcessor_DepositRollback(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Deposit a stackable item
	body := message.DepositBody{
		Slot:          1,
		TemplateId:    2000000,
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   0,
		ReferenceType: string(asset.ReferenceTypeConsumable),
		ReferenceData: message.ReferenceData{
			Quantity: 50,
			OwnerId:  0,
			Flag:     0,
		},
	}

	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		t.Fatalf("Failed to deposit item: %v", err)
	}

	// Rollback the deposit
	rollbackBody := message.DepositRollbackBody{
		AssetId: assetConstants.Id(assetId),
	}

	err = p.DepositRollback(worldId, accountId, rollbackBody)
	if err != nil {
		t.Fatalf("Failed to rollback deposit: %v", err)
	}

	// Verify asset was deleted
	te := tenant.MustFromContext(ctx)
	_, err = asset.GetById(testLogger(), db, te.Id())(assetId)
	if err == nil {
		t.Fatalf("Asset should have been deleted on rollback")
	}

	// Verify stackable was deleted
	_, err = stackable.GetByAssetId(testLogger(), db)(assetId)
	if err == nil {
		t.Fatalf("Stackable should have been deleted on rollback")
	}
}

func TestProcessor_MultipleDeposits(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Deposit multiple items (0-indexed slots)
	for i := 0; i < 3; i++ {
		body := message.DepositBody{
			Slot:          int16(i),
			TemplateId:    uint32(1300000 + i),
			Expiration:    time.Now().Add(time.Hour * 24 * 365),
			ReferenceId:   uint32(i * 100),
			ReferenceType: string(asset.ReferenceTypeEquipable),
		}

		_, err := p.Deposit(worldId, accountId, body)
		if err != nil {
			t.Fatalf("Failed to deposit item %d: %v", i, err)
		}
	}

	// Verify storage has all assets
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to get storage: %v", err)
	}

	te := tenant.MustFromContext(ctx)
	assets, err := asset.GetByStorageId(testLogger(), db, te.Id())(s.Id())
	if err != nil {
		t.Fatalf("Failed to get assets: %v", err)
	}

	if len(assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(assets))
	}
}

func TestProcessor_DeleteByAccountId(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)
	te := tenant.MustFromContext(ctx)

	worldId := world.Id(0)
	accountId := uint32(12345)

	// Create storage and deposit items
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Deposit an equipable item
	body1 := message.DepositBody{
		Slot:          0,
		TemplateId:    1302000,
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   100,
		ReferenceType: string(asset.ReferenceTypeEquipable),
	}
	_, err = p.Deposit(worldId, accountId, body1)
	if err != nil {
		t.Fatalf("Failed to deposit item: %v", err)
	}

	// Deposit a stackable item
	body2 := message.DepositBody{
		Slot:          1,
		TemplateId:    2000000,
		Expiration:    time.Now().Add(time.Hour * 24 * 365),
		ReferenceId:   0,
		ReferenceType: string(asset.ReferenceTypeConsumable),
		ReferenceData: message.ReferenceData{
			Quantity: 50,
			OwnerId:  0,
			Flag:     0,
		},
	}
	_, err = p.Deposit(worldId, accountId, body2)
	if err != nil {
		t.Fatalf("Failed to deposit stackable item: %v", err)
	}

	// Verify storage and assets exist before deletion
	assets, err := asset.GetByStorageId(testLogger(), db, te.Id())(s.Id())
	if err != nil {
		t.Fatalf("Failed to get assets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("Expected 2 assets before deletion, got %d", len(assets))
	}

	// Delete all storage for account
	err = p.DeleteByAccountId(accountId)
	if err != nil {
		t.Fatalf("Failed to delete storage by account ID: %v", err)
	}

	// Verify storage is deleted
	storages, err := storage.GetByAccountId(testLogger(), db, te.Id())(accountId)
	if err != nil {
		t.Fatalf("Failed to query storages: %v", err)
	}
	if len(storages) != 0 {
		t.Fatalf("Expected 0 storages after deletion, got %d", len(storages))
	}
}

func TestProcessor_DeleteByAccountId_MultipleWorlds(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)
	te := tenant.MustFromContext(ctx)

	accountId := uint32(12345)

	// Create storage in multiple worlds
	for worldId := world.Id(0); worldId < 3; worldId++ {
		_, err := p.GetOrCreateStorage(worldId, accountId)
		if err != nil {
			t.Fatalf("Failed to create storage in world %d: %v", worldId, err)
		}

		// Deposit an item in each world
		body := message.DepositBody{
			Slot:          0,
			TemplateId:    1302000 + uint32(worldId),
			Expiration:    time.Now().Add(time.Hour * 24 * 365),
			ReferenceId:   100 + uint32(worldId),
			ReferenceType: string(asset.ReferenceTypeEquipable),
		}
		_, err = p.Deposit(worldId, accountId, body)
		if err != nil {
			t.Fatalf("Failed to deposit item in world %d: %v", worldId, err)
		}
	}

	// Verify storages exist before deletion
	storages, err := storage.GetByAccountId(testLogger(), db, te.Id())(accountId)
	if err != nil {
		t.Fatalf("Failed to query storages: %v", err)
	}
	if len(storages) != 3 {
		t.Fatalf("Expected 3 storages before deletion, got %d", len(storages))
	}

	// Delete all storage for account
	err = p.DeleteByAccountId(accountId)
	if err != nil {
		t.Fatalf("Failed to delete storage by account ID: %v", err)
	}

	// Verify all storages are deleted
	storages, err = storage.GetByAccountId(testLogger(), db, te.Id())(accountId)
	if err != nil {
		t.Fatalf("Failed to query storages: %v", err)
	}
	if len(storages) != 0 {
		t.Fatalf("Expected 0 storages after deletion, got %d", len(storages))
	}
}

func TestProcessor_DeleteByAccountId_NoStorage(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext()
	p := storage.NewProcessor(testLogger(), ctx, db)

	accountId := uint32(99999) // Account with no storage

	// Should not error when deleting non-existent storage
	err := p.DeleteByAccountId(accountId)
	if err != nil {
		t.Fatalf("DeleteByAccountId should not error for non-existent account: %v", err)
	}
}
