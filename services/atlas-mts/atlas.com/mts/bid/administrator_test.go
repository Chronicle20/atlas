package bid_test

import (
	"atlas-mts/bid"
	"atlas-mts/test"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func adminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return test.SetupTestDB(t, bid.Migration)
}

func tenantCtx(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func buildBid(t *testing.T, tenantId uuid.UUID, listingId uuid.UUID, bidderId uint32) bid.Model {
	t.Helper()
	m, err := bid.NewBuilder(tenantId, listingId, bidderId).
		SetAmount(1000).
		SetEscrowTxnId(uuid.New()).
		SetState(bid.StateHeld).
		Build()
	if err != nil {
		t.Fatalf("Failed to build bid: %v", err)
	}
	return m
}

// TestAdministratorCreateGetById asserts a created bid round-trips through
// getById and preserves its fields.
func TestAdministratorCreateGetById(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	listingId := uuid.New()
	escrow := uuid.New()
	m, err := bid.NewBuilder(tenantId, listingId, 100).
		SetAmount(2500).
		SetEscrowTxnId(escrow).
		SetState(bid.StateHeld).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	created, err := bid.CreateBid(db, m)
	if err != nil {
		t.Fatalf("CreateBid: %v", err)
	}
	if created.Id() == uuid.Nil {
		t.Fatal("CreateBid did not assign an id")
	}

	got, err := bid.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.TenantId() != tenantId {
		t.Errorf("tenantId = %s, want %s", got.TenantId(), tenantId)
	}
	if got.ListingId() != listingId {
		t.Errorf("listingId = %s, want %s", got.ListingId(), listingId)
	}
	if got.BidderId() != 100 {
		t.Errorf("bidderId = %d, want 100", got.BidderId())
	}
	if got.Amount() != 2500 {
		t.Errorf("amount = %d, want 2500", got.Amount())
	}
	if got.EscrowTxnId() != escrow {
		t.Errorf("escrowTxnId = %s, want %s", got.EscrowTxnId(), escrow)
	}
	if got.State() != bid.StateHeld {
		t.Errorf("state = %q, want held", got.State())
	}
}

// TestAdministratorCrossTenantIsolation asserts tenant B cannot read tenant A's
// bid.
func TestAdministratorCrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	db := adminTestDB(t)

	dbA := db.WithContext(tenantCtx(t, tenantA))
	dbB := db.WithContext(tenantCtx(t, tenantB))

	created, err := bid.CreateBid(dbA, buildBid(t, tenantA, uuid.New(), 100))
	if err != nil {
		t.Fatalf("CreateBid tenant A: %v", err)
	}

	if _, err := bid.GetById(created.Id().String())(dbA)(); err != nil {
		t.Fatalf("tenant A GetById own bid: %v", err)
	}

	if _, err := bid.GetById(created.Id().String())(dbB)(); err == nil {
		t.Error("tenant B was able to read tenant A's bid")
	}

	allB, err := bid.GetAll()(dbB)()
	if err != nil {
		t.Fatalf("tenant B GetAll: %v", err)
	}
	if len(allB) != 0 {
		t.Errorf("tenant B GetAll returned %d rows, want 0", len(allB))
	}
}

// TestAdministratorUpdateStateConditional asserts the race-safe conditional
// transition: held->released succeeds (1 row), a second held->released affects
// 0 rows.
func TestAdministratorUpdateStateConditional(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	created, err := bid.CreateBid(db, buildBid(t, tenantId, uuid.New(), 100))
	if err != nil {
		t.Fatalf("CreateBid: %v", err)
	}

	affected, err := bid.UpdateState(db, created.Id().String(), bid.StateHeld, bid.StateReleased)
	if err != nil {
		t.Fatalf("UpdateState first: %v", err)
	}
	if affected != 1 {
		t.Errorf("first UpdateState affected %d rows, want 1", affected)
	}

	got, err := bid.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("GetById after transition: %v", err)
	}
	if got.State() != bid.StateReleased {
		t.Errorf("state = %q, want released", got.State())
	}

	affected, err = bid.UpdateState(db, created.Id().String(), bid.StateHeld, bid.StateReleased)
	if err != nil {
		t.Fatalf("UpdateState second: %v", err)
	}
	if affected != 0 {
		t.Errorf("second UpdateState affected %d rows, want 0", affected)
	}
}

// TestAdministratorUpdateStateRejectsMalformedId asserts the nil-id guard: a
// malformed id must ERROR and transition NOTHING, never degrade to a tenant-wide
// state rewrite of every bid in `from` (the GORM zero-value struct-condition
// elision footgun — anti-patterns.md).
func TestAdministratorUpdateStateRejectsMalformedId(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	// Two held bids: a tenant-wide rewrite would flip BOTH.
	a, err := bid.CreateBid(db, buildBid(t, tenantId, uuid.New(), 100))
	if err != nil {
		t.Fatalf("CreateBid a: %v", err)
	}
	b, err := bid.CreateBid(db, buildBid(t, tenantId, uuid.New(), 200))
	if err != nil {
		t.Fatalf("CreateBid b: %v", err)
	}

	affected, err := bid.UpdateState(db, "not-a-uuid", bid.StateHeld, bid.StateReleased)
	if err == nil {
		t.Fatalf("UpdateState with a malformed id must return an error")
	}
	if affected != 0 {
		t.Errorf("malformed-id UpdateState affected %d rows, want 0", affected)
	}

	for _, id := range []string{a.Id().String(), b.Id().String()} {
		got, gerr := bid.GetById(id)(db)()
		if gerr != nil {
			t.Fatalf("GetById %s: %v", id, gerr)
		}
		if got.State() != bid.StateHeld {
			t.Errorf("bid %s state = %q, want held (must not have been rewritten)", id, got.State())
		}
	}
}

// TestAdministratorMultipleBidsPerTenant asserts a single tenant (and a single
// auction) can accept many bids concurrently. Guards against a unique constraint
// on tenant_id alone (which would cap an auction at one bid and break it). The
// (tenant_id, id) unique index must permit this.
func TestAdministratorMultipleBidsPerTenant(t *testing.T) {
	tenantId := uuid.New()
	ctx := tenantCtx(t, tenantId)
	db := adminTestDB(t).WithContext(ctx)

	listingId := uuid.New()
	for i := 0; i < 3; i++ {
		m := buildBid(t, tenantId, listingId, uint32(100+i))
		if _, err := bid.CreateBid(db, m); err != nil {
			t.Fatalf("CreateBid #%d for tenant: %v", i, err)
		}
	}

	all, err := bid.GetAll()(db)()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("tenant holds %d bids, want 3", len(all))
	}
}

// TestAdministratorIndexExists asserts the design index is created by the
// migration.
func TestAdministratorIndexExists(t *testing.T) {
	db := adminTestDB(t)
	mig := db.Migrator()

	if !mig.HasIndex(&bidIndexProbe{}, "idx_bids_listing_state") {
		t.Errorf("expected index %q to exist on bids", "idx_bids_listing_state")
	}
}

type bidIndexProbe struct{}

func (bidIndexProbe) TableName() string { return "bids" }
