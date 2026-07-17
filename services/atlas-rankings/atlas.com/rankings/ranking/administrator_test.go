package ranking

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func testTenantContext(t *testing.T) (tenant.Model, context.Context) {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to register tenant: %v", err)
	}
	return tm, tenant.WithContext(context.Background(), tm)
}

func entityFor(characterId uint32, rank uint32, computedAt time.Time) Entity {
	return Entity{
		CharacterId: characterId,
		WorldId:     world.Id(0),
		JobCategory: 1,
		OverallRank: rank,
		JobRank:     rank,
		ComputedAt:  computedAt,
	}
}

func TestUpsertBatchInsertsAndUpdates(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	t1 := time.Now()
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(101, 1, t1), entityFor(102, 2, t1)}); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	t2 := t1.Add(time.Hour)
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(101, 2, t2), entityFor(102, 1, t2)}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	var count int64
	if err := tdb.Model(&Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows after upsert, got %d", count)
	}

	e, err := byCharacterIdEntityProvider(101)(tdb)()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if e.OverallRank != 2 {
		t.Fatalf("upsert did not update rank: got %d, want 2", e.OverallRank)
	}
	if !e.ComputedAt.Equal(t2) {
		t.Fatalf("upsert did not update computed_at: got %v, want %v", e.ComputedAt, t2)
	}
}

func TestUpsertBatchSpansMultipleBatches(t *testing.T) {
	// upsertBatchSize is 500; seed more than one batch worth of rows in a
	// single call and confirm every row lands, then re-upsert to confirm
	// the ON CONFLICT clause still applies across batch boundaries.
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	const n = 750
	t1 := time.Now()
	seed := make([]Entity, 0, n)
	for i := uint32(0); i < n; i++ {
		seed = append(seed, entityFor(1000+i, i+1, t1))
	}
	if err := upsertBatch(tdb, tm.Id(), seed); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	var count int64
	if err := tdb.Model(&Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != n {
		t.Fatalf("expected %d rows, got %d", n, count)
	}

	t2 := t1.Add(time.Hour)
	reup := make([]Entity, 0, n)
	for i := uint32(0); i < n; i++ {
		reup = append(reup, entityFor(1000+i, n-i, t2))
	}
	if err := upsertBatch(tdb, tm.Id(), reup); err != nil {
		t.Fatalf("re-upsert failed: %v", err)
	}
	if err := tdb.Model(&Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != n {
		t.Fatalf("re-upsert should not grow row count: expected %d, got %d", n, count)
	}

	e, err := byCharacterIdEntityProvider(1000)(tdb)()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if e.OverallRank != n {
		t.Fatalf("last batch's update lost: got rank %d, want %d", e.OverallRank, n)
	}
}

func TestPruneBeforeRemovesStaleRows(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	t1 := time.Now()
	t2 := t1.Add(time.Hour)
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(201, 1, t1)}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(202, 1, t2)}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	if err := pruneBefore(tdb, t2); err != nil {
		t.Fatalf("prune failed: %v", err)
	}

	if _, err := byCharacterIdEntityProvider(201)(tdb)(); err == nil {
		t.Fatal("stale row for character 201 should be pruned")
	}
	if _, err := byCharacterIdEntityProvider(202)(tdb)(); err != nil {
		t.Fatalf("current row for character 202 should survive: %v", err)
	}
}

func TestTenantIsolation(t *testing.T) {
	db := testDatabase(t)
	tmA, ctxA := testTenantContext(t)
	tmB, ctxB := testTenantContext(t)
	dbA := db.WithContext(ctxA)
	dbB := db.WithContext(ctxB)

	now := time.Now()
	if err := upsertBatch(dbA, tmA.Id(), []Entity{entityFor(301, 1, now)}); err != nil {
		t.Fatalf("seed A failed: %v", err)
	}
	if err := upsertBatch(dbB, tmB.Id(), []Entity{entityFor(301, 99, now)}); err != nil {
		t.Fatalf("seed B failed: %v", err)
	}

	// Tenant B cannot read tenant A's row for the same character id — the
	// (tenant_id, character_id) unique index means both rows coexist, and a
	// leaked query would return the wrong tenant's rank (99 vs 1).
	eB, err := byCharacterIdEntityProvider(301)(dbB)()
	if err != nil {
		t.Fatalf("tenant B should read its own row: %v", err)
	}
	if eB.OverallRank != 99 {
		t.Fatalf("tenant B read the wrong row: got rank %d, want 99", eB.OverallRank)
	}

	// allEntityProvider under tenant A must not see tenant B's row.
	allA, err := allEntityProvider()(dbA)()
	if err != nil {
		t.Fatalf("allEntityProvider under A failed: %v", err)
	}
	if len(allA) != 1 || allA[0].OverallRank != 1 {
		t.Fatalf("tenant A's all-rows read leaked tenant B data: %+v", allA)
	}

	// Prune under B, with a cutoff that would also catch A's row by time,
	// must not delete A's row.
	if err := pruneBefore(dbB, now.Add(time.Hour)); err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if _, err := byCharacterIdEntityProvider(301)(dbA)(); err != nil {
		t.Fatalf("tenant A row must survive tenant B prune: %v", err)
	}
	if _, err := byCharacterIdEntityProvider(301)(dbB)(); err == nil {
		t.Fatal("tenant B's own stale row should have been pruned")
	}
}

func TestCycleRows(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	if _, err := cycleEntityProvider()(tdb)(); err == nil {
		t.Fatal("expected no cycle row initially")
	}

	start := time.Now()
	if err := startCycle(tdb, tm.Id(), start); err != nil {
		t.Fatalf("startCycle failed: %v", err)
	}
	if err := completeCycle(tdb, tm.Id(), start.Add(time.Second), 10, 1000); err != nil {
		t.Fatalf("completeCycle failed: %v", err)
	}

	// Second cycle upserts the same row.
	if err := startCycle(tdb, tm.Id(), start.Add(time.Hour)); err != nil {
		t.Fatalf("second startCycle failed: %v", err)
	}
	c, err := cycleEntityProvider()(tdb)()
	if err != nil {
		t.Fatalf("cycle read failed: %v", err)
	}
	if !c.LastStartedAt.After(start) {
		t.Fatalf("second start not recorded: %v", c.LastStartedAt)
	}
	if c.CharactersRanked != 10 {
		t.Fatalf("completion fields lost: %+v", c)
	}

	var count int64
	if err := tdb.Model(&CycleEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 cycle row, got %d", count)
	}
}

func TestCycleRowsTenantIsolation(t *testing.T) {
	db := testDatabase(t)
	tmA, ctxA := testTenantContext(t)
	tmB, ctxB := testTenantContext(t)
	dbA := db.WithContext(ctxA)
	dbB := db.WithContext(ctxB)

	start := time.Now()
	if err := startCycle(dbA, tmA.Id(), start); err != nil {
		t.Fatalf("startCycle A failed: %v", err)
	}
	if err := completeCycle(dbA, tmA.Id(), start.Add(time.Second), 5, 500); err != nil {
		t.Fatalf("completeCycle A failed: %v", err)
	}

	// Tenant B has never run a cycle; it must not see tenant A's row.
	if _, err := cycleEntityProvider()(dbB)(); err == nil {
		t.Fatal("tenant B should have no cycle row yet")
	}

	if err := startCycle(dbB, tmB.Id(), start.Add(time.Minute)); err != nil {
		t.Fatalf("startCycle B failed: %v", err)
	}
	if err := completeCycle(dbB, tmB.Id(), start.Add(time.Minute+time.Second), 7, 700); err != nil {
		t.Fatalf("completeCycle B failed: %v", err)
	}

	cA, err := cycleEntityProvider()(dbA)()
	if err != nil {
		t.Fatalf("tenant A cycle read failed: %v", err)
	}
	if cA.CharactersRanked != 5 {
		t.Fatalf("tenant A cycle row was overwritten by tenant B: %+v", cA)
	}

	cB, err := cycleEntityProvider()(dbB)()
	if err != nil {
		t.Fatalf("tenant B cycle read failed: %v", err)
	}
	if cB.CharactersRanked != 7 {
		t.Fatalf("tenant B cycle row wrong: %+v", cB)
	}

	var count int64
	if err := db.Model(&CycleEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 cycle rows across tenants, got %d", count)
	}
}
