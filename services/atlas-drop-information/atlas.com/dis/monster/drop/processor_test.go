package drop_test

import (
	"atlas-drops-information/monster/drop"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := drop.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func seedTestData(t *testing.T, db *gorm.DB, tenantId uuid.UUID, monsterId uint32, items []uint32) {
	for i, itemId := range items {
		result := db.Exec(
			"INSERT INTO monster_drops (tenant_id, monster_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			tenantId, monsterId, itemId, 1, 5, 0, 50000+uint32(i)*1000,
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

	// Seed test data
	seedTestData(t, db, te.Id(), 100100, []uint32{2000000, 2000001, 2000002})
	seedTestData(t, db, te.Id(), 100101, []uint32{2000003, 2000004})

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
	seedTestData(t, db, te1.Id(), 100100, []uint32{2000000, 2000001})
	// Seed data for tenant 2
	seedTestData(t, db, te2.Id(), 100100, []uint32{2000002, 2000003, 2000004})

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

func TestProcessorImpl_GetForMonster_Success(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	targetMonsterId := uint32(100100)
	otherMonsterId := uint32(100101)

	// Seed test data for target monster
	seedTestData(t, db, te.Id(), targetMonsterId, []uint32{2000000, 2000001, 2000002})
	// Seed test data for other monster
	seedTestData(t, db, te.Id(), otherMonsterId, []uint32{2000003, 2000004})

	p := drop.NewProcessor(l, ctx, db)

	results, err := p.GetForMonster(targetMonsterId)()
	if err != nil {
		t.Fatalf("GetForMonster() returned error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 drops for monster %d, got %d", targetMonsterId, len(results))
	}

	// Verify all results are for the correct monster
	for _, r := range results {
		if r.MonsterId() != targetMonsterId {
			t.Errorf("Expected MonsterId %d, got %d", targetMonsterId, r.MonsterId())
		}
	}
}

func TestProcessorImpl_GetForMonster_Empty(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	// Seed data for a different monster
	seedTestData(t, db, te.Id(), 100100, []uint32{2000000})

	p := drop.NewProcessor(l, ctx, db)

	// Query for non-existent monster
	results, err := p.GetForMonster(999999)()
	if err != nil {
		t.Fatalf("GetForMonster() returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 drops for non-existent monster, got %d", len(results))
	}
}

func TestProcessorImpl_GetForMonster_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	te1 := testTenant()
	te2 := testTenant()
	ctx1 := tenant.WithContext(context.Background(), te1)
	db := testDatabase(t)

	monsterId := uint32(100100)

	// Seed data for both tenants with same monster
	seedTestData(t, db, te1.Id(), monsterId, []uint32{2000000, 2000001})
	seedTestData(t, db, te2.Id(), monsterId, []uint32{2000002, 2000003, 2000004})

	p := drop.NewProcessor(l, ctx1, db)

	results, err := p.GetForMonster(monsterId)()
	if err != nil {
		t.Fatalf("GetForMonster() returned error: %v", err)
	}

	// Should only return tenant 1's data
	if len(results) != 2 {
		t.Errorf("Expected 2 drops for tenant 1, got %d", len(results))
	}
}

func TestProcessorImpl_GetForMonster_VerifyFields(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	monsterId := uint32(100100)
	itemId := uint32(2000000)

	// Seed a specific drop with known values
	result := db.Exec(
		"INSERT INTO monster_drops (tenant_id, monster_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
		te.Id(), monsterId, itemId, 1, 10, 1001, 75000,
	)
	if result.Error != nil {
		t.Fatalf("Failed to seed test data: %v", result.Error)
	}

	p := drop.NewProcessor(l, ctx, db)

	results, err := p.GetForMonster(monsterId)()
	if err != nil {
		t.Fatalf("GetForMonster() returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 drop, got %d", len(results))
	}

	d := results[0]
	if d.TenantId() != te.Id() {
		t.Errorf("Expected TenantId %s, got %s", te.Id(), d.TenantId())
	}
	if d.MonsterId() != monsterId {
		t.Errorf("Expected MonsterId %d, got %d", monsterId, d.MonsterId())
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
