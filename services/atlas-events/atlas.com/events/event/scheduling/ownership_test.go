package scheduling

// ownership_test.go — task-232 (sparse ephemeral environments) applied to the
// one deliberately cross-tenant reader in this service.
//
// The poller sees every tenant's due work by design (§4.2). Once a PR
// environment shares this database with the baseline deployment, "every
// tenant" includes tenants the pod does not serve, and the two failure modes
// below are what ProcessorImpl.owns exists to prevent. These tests drive the
// predicate directly rather than through libs/atlas-env, because the
// predicate is injected from main.go precisely so this package never imports
// it (NG5/FR-4.5).

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// insertWorkForTenant writes a due PENDING row owned by tenantId, bypassing
// insertWork's context-derived tenant so a test can stage a row belonging to
// an environment this deployment does not serve.
func insertWorkForTenant(t *testing.T, db *gorm.DB, tenantId uuid.UUID, state string, executeAt time.Time) uuid.UUID {
	t.Helper()
	tm, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	m, err := NewBuilder(uuid.New(), WorkTypeTriggerEvaluation).
		SetExecuteAt(executeAt).
		SetState(state).
		Build()
	if err != nil {
		t.Fatalf("build work: %v", err)
	}
	e, err := ToEntity(m, tm)
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("insert work: %v", err)
	}
	return e.ID
}

// ownsOnly builds a TenantOwnership admitting exactly one tenant.
func ownsOnly(id uuid.UUID) TenantOwnership {
	return func(tenantId uuid.UUID) bool { return tenantId == id }
}

// A row belonging to an environment this deployment does not serve must not
// be claimed — and, critically, must be left PENDING rather than parked in
// PROCESSING under this pod's instanceId, which would delay the pod that DOES
// own it by a full lease interval.
func TestClaimBatchSkipsUnownedTenantAndLeavesItPending(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)

	mine := uuid.New()
	theirs := uuid.New()
	mineId := insertWorkForTenant(t, db, mine, StatePending, time.Now().Add(-time.Hour))
	theirsId := insertWorkForTenant(t, db, theirs, StatePending, time.Now().Add(-time.Hour))

	p := NewProcessor(testLogger(t), ctx, db)
	p.SetOwnership(ownsOnly(mine))

	claimed, err := p.ClaimBatch("baseline", 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed %d rows, want 1 (only the owned tenant's)", len(claimed))
	}
	if claimed[0].Id() != mineId {
		t.Fatalf("claimed row %s, want the owned tenant's row %s", claimed[0].Id(), mineId)
	}

	if got := readWork(t, db, theirsId); got.State != StatePending {
		t.Fatalf("unowned row state = %q, want %q — it was claimed and stranded", got.State, StatePending)
	}
	if got := readWork(t, db, theirsId); got.ClaimedBy != "" {
		t.Fatalf("unowned row claimed_by = %q, want empty", got.ClaimedBy)
	}
}

// With no predicate installed the poller keeps its pre-task-232 behaviour:
// every visible tenant is this deployment's. Every other test in this package
// relies on that default, so it is pinned explicitly.
func TestClaimBatchWithoutOwnershipClaimsEveryTenant(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)

	insertWorkForTenant(t, db, uuid.New(), StatePending, time.Now().Add(-time.Hour))
	insertWorkForTenant(t, db, uuid.New(), StatePending, time.Now().Add(-time.Hour))

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("baseline", 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed %d rows, want 2 (no ownership predicate means own everything)", len(claimed))
	}
}

// The lease exists to detect a DEAD claimer, not a claimer in another
// environment. Reclaiming an unowned row would bump its attempts and hand it
// back to PENDING while the pod that owns it is still working on it.
func TestReclaimSkipsUnownedTenant(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)

	mine := uuid.New()
	theirs := uuid.New()
	stale := time.Now().Add(-time.Hour)

	mineId := insertWorkForTenant(t, db, mine, StateProcessing, time.Now())
	theirsId := insertWorkForTenant(t, db, theirs, StateProcessing, time.Now())
	for _, id := range []uuid.UUID{mineId, theirsId} {
		if err := db.Model(&Entity{}).Where("id = ?", id).
			Updates(map[string]any{"claimed_by": "dead", "claimed_at": stale}).Error; err != nil {
			t.Fatalf("mark claimed: %v", err)
		}
	}

	p := NewProcessor(testLogger(t), ctx, db)
	p.SetOwnership(ownsOnly(mine))

	n, err := p.Reclaim(time.Minute)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 1 {
		t.Fatalf("reclaimed %d rows, want 1 (only the owned tenant's)", n)
	}

	if got := readWork(t, db, mineId); got.State != StatePending || got.Attempts != 1 {
		t.Fatalf("owned row = (%q, attempts %d), want (%q, 1)", got.State, got.Attempts, StatePending)
	}
	got := readWork(t, db, theirsId)
	if got.State != StateProcessing {
		t.Fatalf("unowned row state = %q, want %q — another environment's in-flight row was reset", got.State, StateProcessing)
	}
	if got.Attempts != 0 {
		t.Fatalf("unowned row attempts = %d, want 0 — its retry budget was consumed by the wrong deployment", got.Attempts)
	}
	if got.ClaimedBy != "dead" {
		t.Fatalf("unowned row claimed_by = %q, want %q", got.ClaimedBy, "dead")
	}
}

// Reclaim's owns-everything path (nil predicate) must stay the single bulk
// UPDATE it has always been, resetting every stale row regardless of tenant.
func TestReclaimWithoutOwnershipReclaimsEveryTenant(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)

	stale := time.Now().Add(-time.Hour)
	for i := 0; i < 2; i++ {
		id := insertWorkForTenant(t, db, uuid.New(), StateProcessing, time.Now())
		if err := db.Model(&Entity{}).Where("id = ?", id).
			Updates(map[string]any{"claimed_by": "dead", "claimed_at": stale}).Error; err != nil {
			t.Fatalf("mark claimed: %v", err)
		}
	}

	p := NewProcessor(testLogger(t), ctx, db)
	n, err := p.Reclaim(time.Minute)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 2 {
		t.Fatalf("reclaimed %d rows, want 2 (no ownership predicate means own everything)", n)
	}
}
