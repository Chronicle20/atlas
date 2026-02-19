package drop_test

import (
	"atlas-drops-information/continent/drop"
	"context"
	"testing"

	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

	if err := drop.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func seedTestData(t *testing.T, db *gorm.DB, tenantId uuid.UUID, continentId int32, items []uint32) {
	for i, itemId := range items {
		result := db.Exec(
			"INSERT INTO continent_drops (tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			tenantId, continentId, itemId, 1, 5, 0, 50000+uint32(i)*1000,
		)
		if result.Error != nil {
			t.Fatalf("Failed to seed test data: %v", result.Error)
		}
	}
}

func TestProcessorImpl_GetAll_Success(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	// Seed test data for multiple continents
	seedTestData(t, db, te.Id(), 0, []uint32{2000000, 2000001, 2000002})  // Victoria Island
	seedTestData(t, db, te.Id(), 1, []uint32{2000003, 2000004})           // Ossyria

	p := drop.NewProcessor(l, ctx, db)

	results, err := p.GetAll()()
	if err != nil {
		t.Fatalf("GetAll() returned error: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 drops, got %d", len(results))
	}
}

func TestProcessorImpl_GetAll_Empty(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := drop.NewProcessor(l, ctx, db)

	results, err := p.GetAll()()
	if err != nil {
		t.Fatalf("GetAll() returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 drops, got %d", len(results))
	}
}

func TestProcessorImpl_GetAll_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	te1 := testTenant()
	te2 := testTenant()
	ctx1 := tenant.WithContext(context.Background(), te1)
	db := testDatabase(t)

	// Seed data for tenant 1
	seedTestData(t, db, te1.Id(), 0, []uint32{2000000, 2000001})
	// Seed data for tenant 2
	seedTestData(t, db, te2.Id(), 0, []uint32{2000002, 2000003, 2000004})

	p := drop.NewProcessor(l, ctx1, db)

	results, err := p.GetAll()()
	if err != nil {
		t.Fatalf("GetAll() returned error: %v", err)
	}

	// Should only return tenant 1's data
	if len(results) != 2 {
		t.Errorf("Expected 2 drops for tenant 1, got %d", len(results))
	}
}

func TestProcessorImpl_GetAll_VerifyFields(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	continentId := int32(0)
	itemId := uint32(2000000)

	// Seed a specific drop with known values
	result := db.Exec(
		"INSERT INTO continent_drops (tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
		te.Id(), continentId, itemId, 1, 10, 1001, 75000,
	)
	if result.Error != nil {
		t.Fatalf("Failed to seed test data: %v", result.Error)
	}

	p := drop.NewProcessor(l, ctx, db)

	results, err := p.GetAll()()
	if err != nil {
		t.Fatalf("GetAll() returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 drop, got %d", len(results))
	}

	d := results[0]
	if d.TenantId() != te.Id() {
		t.Errorf("Expected TenantId %s, got %s", te.Id(), d.TenantId())
	}
	if d.ContinentId() != continentId {
		t.Errorf("Expected ContinentId %d, got %d", continentId, d.ContinentId())
	}
	if d.ItemId() != itemId {
		t.Errorf("Expected ItemId %d, got %d", itemId, d.ItemId())
	}
	if d.MinimumQuantity() != 1 {
		t.Errorf("Expected MinimumQuantity 1, got %d", d.MinimumQuantity())
	}
	if d.MaximumQuantity() != 10 {
		t.Errorf("Expected MaximumQuantity 10, got %d", d.MaximumQuantity())
	}
	if d.QuestId() != 1001 {
		t.Errorf("Expected QuestId 1001, got %d", d.QuestId())
	}
	if d.Chance() != 75000 {
		t.Errorf("Expected Chance 75000, got %d", d.Chance())
	}
}

func TestProcessorImpl_GetAll_MultipleContinents(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	// Seed data for multiple continents
	seedTestData(t, db, te.Id(), 0, []uint32{2000000})   // Victoria Island
	seedTestData(t, db, te.Id(), 1, []uint32{2000001})   // Ossyria
	seedTestData(t, db, te.Id(), 2, []uint32{2000002})   // Ludus Lake

	p := drop.NewProcessor(l, ctx, db)

	results, err := p.GetAll()()
	if err != nil {
		t.Fatalf("GetAll() returned error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 drops from 3 continents, got %d", len(results))
	}

	// Verify we got drops from different continents
	continentIds := make(map[int32]bool)
	for _, d := range results {
		continentIds[d.ContinentId()] = true
	}

	if len(continentIds) != 3 {
		t.Errorf("Expected drops from 3 different continents, got %d", len(continentIds))
	}
}
