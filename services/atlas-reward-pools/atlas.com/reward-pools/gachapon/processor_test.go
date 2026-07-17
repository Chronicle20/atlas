package gachapon_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/test"
	"context"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func countTestDatabase(t *testing.T) *gorm.DB {
	l := logrus.New()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := gachapon.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func countTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

func seedCountGachapon(t *testing.T, p gachapon.Processor, tenantId uuid.UUID, id string) {
	t.Helper()
	m, err := gachapon.NewBuilder(tenantId, id).
		SetName("count-" + id).
		SetNpcIds([]uint32{9100100}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon %s: %v", id, err)
	}
	if err := p.Create(m); err != nil {
		t.Fatalf("Failed to create gachapon %s: %v", id, err)
	}
}

func TestProcessorImpl_Count_Empty(t *testing.T) {
	l := logrus.New()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := countTestDatabase(t)

	p := gachapon.NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
	if updated != nil {
		t.Errorf("Expected nil updatedAt, got %v", updated)
	}
}

func TestProcessorImpl_Count_Populated(t *testing.T) {
	l := logrus.New()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := countTestDatabase(t)

	p := gachapon.NewProcessor(l, ctx, db)
	seedCountGachapon(t, p, te.Id(), "count-pop-1")
	seedCountGachapon(t, p, te.Id(), "count-pop-2")
	seedCountGachapon(t, p, te.Id(), "count-pop-3")

	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	// gachapons has no updated_at column; updatedAt must be nil.
	if updated != nil {
		t.Errorf("Expected nil updatedAt for table without updated_at, got %v", updated)
	}
}

func TestProcessorImpl_Create_CrossTenantSameSlug(t *testing.T) {
	l := logrus.New()
	te1 := countTestTenant(t)
	te2 := countTestTenant(t)
	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)
	db := countTestDatabase(t)

	p1 := gachapon.NewProcessor(l, ctx1, db)
	p2 := gachapon.NewProcessor(l, ctx2, db)

	// Regression: the gachapons primary key was the slug ("id") alone, not
	// scoped by tenant. Gachapon slugs are identical across tenants, so the
	// second tenant to seed a slug collided on the primary key and got zero
	// rows. Each tenant must be able to own its own copy of a slug.
	seedCountGachapon(t, p1, te1.Id(), "henesys")
	seedCountGachapon(t, p2, te2.Id(), "henesys")

	g1, err := p1.GetById("henesys")
	if err != nil {
		t.Fatalf("tenant 1 GetById(henesys): %v", err)
	}
	if g1.TenantId() != te1.Id() {
		t.Errorf("tenant 1 row tenant = %s, want %s", g1.TenantId(), te1.Id())
	}
	g2, err := p2.GetById("henesys")
	if err != nil {
		t.Fatalf("tenant 2 GetById(henesys): %v", err)
	}
	if g2.TenantId() != te2.Id() {
		t.Errorf("tenant 2 row tenant = %s, want %s", g2.TenantId(), te2.Id())
	}

	c1, _, err := p1.Count()
	if err != nil {
		t.Fatalf("tenant 1 Count: %v", err)
	}
	c2, _, err := p2.Count()
	if err != nil {
		t.Fatalf("tenant 2 Count: %v", err)
	}
	if c1 != 1 || c2 != 1 {
		t.Errorf("per-tenant counts = %d, %d; want 1, 1", c1, c2)
	}

	// Slug-based Update must not reach across tenants. With a shared slug this
	// is only correct because the tenant callback scopes the write; assert it.
	if err := p1.Update("henesys", "renamed-by-1", []uint32{9100100}, 60, 30, 10); err != nil {
		t.Fatalf("tenant 1 Update(henesys): %v", err)
	}
	g2After, err := p2.GetById("henesys")
	if err != nil {
		t.Fatalf("tenant 2 GetById(henesys) after tenant 1 update: %v", err)
	}
	if g2After.Name() != "count-henesys" {
		t.Errorf("tenant 1 Update leaked into tenant 2: name = %q, want %q", g2After.Name(), "count-henesys")
	}

	// Slug-based Delete must not reach across tenants either.
	if err := p1.Delete("henesys"); err != nil {
		t.Fatalf("tenant 1 Delete(henesys): %v", err)
	}
	if _, err := p2.GetById("henesys"); err != nil {
		t.Errorf("tenant 1 Delete removed tenant 2's row: %v", err)
	}
	if _, err := p1.GetById("henesys"); err == nil {
		t.Error("tenant 1 Delete did not remove tenant 1's own row")
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l := logrus.New()
	te1 := countTestTenant(t)
	te2 := countTestTenant(t)
	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)
	db := countTestDatabase(t)

	p1 := gachapon.NewProcessor(l, ctx1, db)
	p2 := gachapon.NewProcessor(l, ctx2, db)

	// Tenant 1: 2 rows
	seedCountGachapon(t, p1, te1.Id(), "count-iso-1a")
	seedCountGachapon(t, p1, te1.Id(), "count-iso-1b")
	// Tenant 2: 3 rows
	seedCountGachapon(t, p2, te2.Id(), "count-iso-2a")
	seedCountGachapon(t, p2, te2.Id(), "count-iso-2b")
	seedCountGachapon(t, p2, te2.Id(), "count-iso-2c")

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}

func TestGachaponProcessorCRUD(t *testing.T) {
	processor, db, cleanup := test.CreateGachaponProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	t.Run("Create and GetById", func(t *testing.T) {
		// Create a gachapon
		model, err := gachapon.NewBuilder(tenantId, "crud-test-1").
			SetName("CRUD Test Gachapon").
			SetNpcIds([]uint32{9100100, 9100101}).
			SetCommonWeight(70).
			SetUncommonWeight(25).
			SetRareWeight(5).
			Build()
		if err != nil {
			t.Fatalf("Failed to build gachapon: %v", err)
		}

		err = processor.Create(model)
		if err != nil {
			t.Fatalf("Failed to create gachapon: %v", err)
		}

		// Get by ID
		retrieved, err := processor.GetById("crud-test-1")
		if err != nil {
			t.Fatalf("Failed to get gachapon by ID: %v", err)
		}

		if retrieved.Id() != "crud-test-1" {
			t.Errorf("Expected ID 'crud-test-1', got '%s'", retrieved.Id())
		}
		if retrieved.Name() != "CRUD Test Gachapon" {
			t.Errorf("Expected name 'CRUD Test Gachapon', got '%s'", retrieved.Name())
		}
		if len(retrieved.NpcIds()) != 2 {
			t.Errorf("Expected 2 NPC IDs, got %d", len(retrieved.NpcIds()))
		}
		if retrieved.CommonWeight() != 70 {
			t.Errorf("Expected common weight 70, got %d", retrieved.CommonWeight())
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create another gachapon
		model, err := gachapon.NewBuilder(tenantId, "crud-test-2").
			SetName("Second Gachapon").
			SetNpcIds([]uint32{9100102}).
			SetCommonWeight(50).
			SetUncommonWeight(40).
			SetRareWeight(10).
			Build()
		if err != nil {
			t.Fatalf("Failed to build gachapon: %v", err)
		}

		err = processor.Create(model)
		if err != nil {
			t.Fatalf("Failed to create second gachapon: %v", err)
		}

		// Get all
		paged, err := processor.GetAll(atlasmodel.Page{Number: 1, Size: 50})()
		if err != nil {
			t.Fatalf("Failed to get all gachapons: %v", err)
		}

		if len(paged.Items) < 2 {
			t.Errorf("Expected at least 2 gachapons, got %d", len(paged.Items))
		}
	})

	t.Run("Update", func(t *testing.T) {
		// Update the first gachapon
		err := processor.Update("crud-test-1", "Updated Name", []uint32{9100100, 9100101}, 60, 30, 10)
		if err != nil {
			t.Fatalf("Failed to update gachapon: %v", err)
		}

		// Verify update
		updated, err := processor.GetById("crud-test-1")
		if err != nil {
			t.Fatalf("Failed to get updated gachapon: %v", err)
		}

		if updated.Name() != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name())
		}
		if updated.CommonWeight() != 60 {
			t.Errorf("Expected common weight 60, got %d", updated.CommonWeight())
		}
		if updated.UncommonWeight() != 30 {
			t.Errorf("Expected uncommon weight 30, got %d", updated.UncommonWeight())
		}
		if updated.RareWeight() != 10 {
			t.Errorf("Expected rare weight 10, got %d", updated.RareWeight())
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// Delete the second gachapon
		err := processor.Delete("crud-test-2")
		if err != nil {
			t.Fatalf("Failed to delete gachapon: %v", err)
		}

		// Verify deletion
		_, err = processor.GetById("crud-test-2")
		if err == nil {
			t.Error("Expected error when getting deleted gachapon, got nil")
		}
	})

	t.Run("GetById NotFound", func(t *testing.T) {
		_, err := processor.GetById("non-existent")
		if err == nil {
			t.Error("Expected error when getting non-existent gachapon, got nil")
		}
	})

	t.Run("BulkCreate", func(t *testing.T) {
		// Create multiple gachapons
		models := make([]gachapon.Model, 3)
		for i := 0; i < 3; i++ {
			m, err := gachapon.NewBuilder(tenantId, "bulk-test-"+string(rune('A'+i))).
				SetName("Bulk Gachapon " + string(rune('A'+i))).
				SetNpcIds([]uint32{uint32(9100200 + i)}).
				SetCommonWeight(70).
				SetUncommonWeight(25).
				SetRareWeight(5).
				Build()
			if err != nil {
				t.Fatalf("Failed to build bulk gachapon %d: %v", i, err)
			}
			models[i] = m
		}

		err := gachapon.BulkCreateGachapon(db, models)
		if err != nil {
			t.Fatalf("Failed to bulk create gachapons: %v", err)
		}

		// Verify all were created
		for i := 0; i < 3; i++ {
			_, err := processor.GetById("bulk-test-" + string(rune('A'+i)))
			if err != nil {
				t.Errorf("Failed to get bulk-created gachapon %d: %v", i, err)
			}
		}
	})
}
