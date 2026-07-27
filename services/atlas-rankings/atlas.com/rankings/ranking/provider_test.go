package ranking

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestByWorldPagedOrdersAndFilters(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)
	seed := []Entity{
		{CharacterId: 1, WorldId: 0, JobCategory: 1, OverallRank: 2, JobRank: 1, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 2, WorldId: 0, JobCategory: 1, OverallRank: 1, JobRank: 2, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 3, WorldId: 1, JobCategory: 1, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	}
	if err := upsertBatch(tdb, tm.Id(), seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	page := model.Page{Number: 1, Size: 10}
	paged, err := byWorldPagedEntityProvider(0, nil, page)(tdb)()
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if paged.Total != 2 {
		t.Fatalf("Total = %d, want 2 (world 0 only)", paged.Total)
	}
	if len(paged.Items) != 2 || paged.Items[0].CharacterId != 2 || paged.Items[1].CharacterId != 1 {
		t.Fatalf("overall order wrong: %+v", paged.Items)
	}
}

func TestByWorldPagedCategoryFilter(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)
	cat := uint16(1)
	seed := []Entity{
		{CharacterId: 1, WorldId: 0, JobCategory: 1, OverallRank: 3, JobRank: 2, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 2, WorldId: 0, JobCategory: 1, OverallRank: 4, JobRank: 1, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 3, WorldId: 0, JobCategory: 2, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	}
	if err := upsertBatch(tdb, tm.Id(), seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	paged, err := byWorldPagedEntityProvider(0, &cat, model.Page{Number: 1, Size: 10})(tdb)()
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if paged.Total != 2 || len(paged.Items) != 2 || paged.Items[0].CharacterId != 2 || paged.Items[1].CharacterId != 1 {
		t.Fatalf("category filter/order wrong: %+v", paged.Items)
	}
}

// TestByWorldPagedTenantIsolation proves the provider relies on the
// context-bearing db handle's tenant callback rather than adding its own
// tenant filter — tenant B must not see tenant A's world-0 rows even though
// both tenants use the same world id.
func TestByWorldPagedTenantIsolation(t *testing.T) {
	db := testDatabase(t)
	tmA, ctxA := testTenantContext(t)
	tmB, ctxB := testTenantContext(t)
	tdbA := db.WithContext(ctxA)
	tdbB := db.WithContext(ctxB)

	if err := upsertBatch(tdbA, tmA.Id(), []Entity{
		{CharacterId: 1, WorldId: 0, JobCategory: 1, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	}); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if err := upsertBatch(tdbB, tmB.Id(), []Entity{
		{CharacterId: 2, WorldId: 0, JobCategory: 1, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	paged, err := byWorldPagedEntityProvider(0, nil, model.Page{Number: 1, Size: 10})(tdbA)()
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if paged.Total != 1 || len(paged.Items) != 1 || paged.Items[0].CharacterId != 1 {
		t.Fatalf("tenant A leaked tenant B rows: %+v", paged)
	}
}
