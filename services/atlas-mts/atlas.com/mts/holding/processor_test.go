package holding_test

import (
	"atlas-mts/holding"
	"atlas-mts/test"
	"testing"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// resetHoldings clears the holdings table. The shared in-memory SQLite DB is
// reused across tests in the process, and these processor tests all run under
// the fixed test tenant, so rows from prior tests would otherwise leak into
// GetByOwner/GetAll counts. Truncating up front makes each processor test
// deterministic regardless of execution order.
func resetHoldings(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset holdings: %v", err)
	}
}

// buildProcessorHolding builds a holding for the test tenant. The tenant id MUST
// match the processor's context tenant (test.TestTenantId) so the row is visible
// through the processor's tenant-scoped queries.
func buildProcessorHolding(t *testing.T, worldId world.Id, ownerId uint32) holding.Model {
	t.Helper()
	m, err := holding.NewBuilder(test.TestTenantId, worldId, ownerId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("Failed to build holding: %v", err)
	}
	return m
}

// TestProcessorCreateGetById asserts a created holding round-trips through the
// processor's GetById.
func TestProcessorCreateGetById(t *testing.T) {
	p, db, cleanup := test.CreateHoldingProcessor(t)
	defer cleanup()
	resetHoldings(t, db)

	created, err := p.Create(buildProcessorHolding(t, 0, 100))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Id().String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("Create did not assign an id")
	}

	got, err := p.GetById(created.Id().String())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.OwnerId() != 100 {
		t.Errorf("ownerId = %d, want 100", got.OwnerId())
	}
	if got.Origin() != holding.OriginPurchased {
		t.Errorf("origin = %q, want purchased", got.Origin())
	}
}

// TestProcessorGetByOwner asserts GetByOwner filters by world and owner.
func TestProcessorGetByOwner(t *testing.T) {
	p, db, cleanup := test.CreateHoldingProcessor(t)
	defer cleanup()
	resetHoldings(t, db)

	// world 0, owner 100 (x2)
	if _, err := p.Create(buildProcessorHolding(t, 0, 100)); err != nil {
		t.Fatalf("Create w0 o100 #1: %v", err)
	}
	if _, err := p.Create(buildProcessorHolding(t, 0, 100)); err != nil {
		t.Fatalf("Create w0 o100 #2: %v", err)
	}
	// world 0, owner 101
	if _, err := p.Create(buildProcessorHolding(t, 0, 101)); err != nil {
		t.Fatalf("Create w0 o101: %v", err)
	}
	// world 1, owner 100
	if _, err := p.Create(buildProcessorHolding(t, 1, 100)); err != nil {
		t.Fatalf("Create w1 o100: %v", err)
	}

	// GetByOwner(world 0, owner 100) => exactly 2 rows.
	got, err := p.GetByOwner(0, 100)
	if err != nil {
		t.Fatalf("GetByOwner: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetByOwner(w0, o100) returned %d rows, want 2", len(got))
	}
	for _, h := range got {
		if h.WorldId() != 0 || h.OwnerId() != 100 {
			t.Errorf("GetByOwner returned wrong row: world=%d owner=%d", h.WorldId(), h.OwnerId())
		}
	}
}

// TakeHome's saga-building behavior (it builds + emits a WithdrawFromMts saga and
// does NOT directly soft-delete the holding) is asserted in take_home_flow_test.go
// (TestTakeHomeBuildsWithdrawFromMtsSaga). The replay no-op (idempotency) is
// enforced at the custody layer by the ReleaseFromMtsHolding handler — see
// TestReleaseFromMtsHolding_SoftDeletesAndIsIdempotent in the custody consumer.
