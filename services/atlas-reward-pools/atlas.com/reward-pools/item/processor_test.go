package item_test

import (
	"atlas-reward-pools/item"
	"context"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
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
	if err := item.Migration(db); err != nil {
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

func seedCountItem(t *testing.T, p item.Processor, tenantId uuid.UUID, gachaponId string, itemId uint32) {
	t.Helper()
	m, err := item.NewBuilder(tenantId, 0).
		SetGachaponId(gachaponId).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build item: %v", err)
	}
	if err := p.Create(m); err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}
}

func TestProcessorImpl_Count_Empty(t *testing.T) {
	l := logrus.New()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := countTestDatabase(t)

	p := item.NewProcessor(l, ctx, db)
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

	p := item.NewProcessor(l, ctx, db)
	seedCountItem(t, p, te.Id(), "gacha-pop-1", 2000000)
	seedCountItem(t, p, te.Id(), "gacha-pop-1", 2000001)
	seedCountItem(t, p, te.Id(), "gacha-pop-2", 2000002)

	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	// gachapon_items has no updated_at column; updatedAt must be nil.
	if updated != nil {
		t.Errorf("Expected nil updatedAt for table without updated_at, got %v", updated)
	}
}

// TestProcessorImpl_GetByGachaponId_AllTiers seeds two items for the same
// gachaponId across different tiers and verifies GetByGachaponId returns
// both regardless of tier — the existing tier-scoped roll
// (GetByGachaponIdAndTier) is unaffected by this method.
func TestProcessorImpl_GetByGachaponId_AllTiers(t *testing.T) {
	l := logrus.New()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := countTestDatabase(t)

	p := item.NewProcessor(l, ctx, db)

	commonItem, err := item.NewBuilder(te.Id(), 0).
		SetGachaponId("gacha-multi-tier").
		SetItemId(4000000).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build common item: %v", err)
	}
	if err := p.Create(commonItem); err != nil {
		t.Fatalf("Failed to create common item: %v", err)
	}

	rareItem, err := item.NewBuilder(te.Id(), 0).
		SetGachaponId("gacha-multi-tier").
		SetItemId(4000001).
		SetQuantity(1).
		SetTier("rare").
		Build()
	if err != nil {
		t.Fatalf("Failed to build rare item: %v", err)
	}
	if err := p.Create(rareItem); err != nil {
		t.Fatalf("Failed to create rare item: %v", err)
	}

	items, err := p.GetByGachaponId("gacha-multi-tier")()
	if err != nil {
		t.Fatalf("GetByGachaponId() returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 items across tiers, got %d", len(items))
	}

	tiers := make(map[string]bool)
	for _, it := range items {
		tiers[it.Tier()] = true
	}
	if !tiers["common"] {
		t.Errorf("Expected common-tier item in result, tiers seen: %v", tiers)
	}
	if !tiers["rare"] {
		t.Errorf("Expected rare-tier item in result, tiers seen: %v", tiers)
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l := logrus.New()
	te1 := countTestTenant(t)
	te2 := countTestTenant(t)
	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)
	db := countTestDatabase(t)

	p1 := item.NewProcessor(l, ctx1, db)
	p2 := item.NewProcessor(l, ctx2, db)

	// Tenant 1: 2 rows
	seedCountItem(t, p1, te1.Id(), "gacha-iso-1", 3000000)
	seedCountItem(t, p1, te1.Id(), "gacha-iso-1", 3000001)
	// Tenant 2: 3 rows
	seedCountItem(t, p2, te2.Id(), "gacha-iso-2", 3000002)
	seedCountItem(t, p2, te2.Id(), "gacha-iso-2", 3000003)
	seedCountItem(t, p2, te2.Id(), "gacha-iso-2", 3000004)

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}
