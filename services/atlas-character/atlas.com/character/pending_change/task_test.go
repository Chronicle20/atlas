package pending_change

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// seedCharacterInContext mirrors refund_idempotency_test.go's seedCharacter
// but takes an explicit context, so multi-tenant tests can seed one
// character per tenant under that tenant's own context rather than the
// package's single fixed testTenantModel.
func seedCharacterInContext(t *testing.T, ctx context.Context, db *gorm.DB, name string, worldId world.Id) uint32 {
	t.Helper()
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(worldId).
		SetName(name).
		SetLevel(1).
		SetExperience(0).
		Build()
	c, err := character.NewProcessor(testLogger(t), ctx, db).
		Create(message.NewBuffer())(uuid.New(), input, 0)
	if err != nil {
		t.Fatalf("seed character %s: %v", name, err)
	}
	return c.Id()
}

// TestSweepExpiresAndRefundsAPastDueRequest is the plan's canonical fixture:
// a request whose ExpiresAt has been backdated is swept to EXPIRED and
// refunded exactly once, even across two Sweep calls. Uses
// newProcessorTestDB/seedCharacter (refund_idempotency_test.go) rather than
// a bare Migration(db), because CreateAndEmit looks the character up by id
// before it will create a pending change.
func TestSweepExpiresAndRefundsAPastDueRequest(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	p := NewProcessor(l, ctx, db)
	characterId := seedCharacter(t, db, "Quebec", world.Id(0))

	assetId := uint32(9003)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Quebectwo", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	// Backdate expires_at rather than sleeping.
	if err := db.Model(&entity{}).Where("id = ?", m.Id()).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}

	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	got, _ := p.GetById(m.Id())
	if got.Status() != StatusExpired || got.Reason() != "expired" {
		t.Fatalf("got %s / %s, want EXPIRED / expired", got.Status(), got.Reason())
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("expected exactly one refund")
	}

	// Idempotent: a second sweep must not refund again.
	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("second Sweep: %v", err)
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("second sweep refunded again")
	}
}

// TestSweepLeavesNotYetDueRequestPending proves Sweep only touches rows
// whose deadline has actually passed -- a request created moments ago (real
// ExpiresAt, no backdating) must survive a sweep untouched and unrefunded.
func TestSweepLeavesNotYetDueRequestPending(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	p := NewProcessor(l, ctx, db)
	characterId := seedCharacter(t, db, "Romeo", world.Id(0))

	assetId := uint32(9004)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Romeotwo", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	got, err := p.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != StatusPending {
		t.Fatalf("got status %s, want PENDING", got.Status())
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 0 {
		t.Fatal("not-yet-due request produced a refund")
	}
}

// TestSweepDoesNotReSweepATerminalRequest proves the terminal-state guard:
// a request already resolved to a terminal status (CANCELLED here, not via
// the sweep path) is never picked up by a later sweep even once its
// ExpiresAt is backdated into the past -- getExpired filters on
// status = PENDING, so a terminal row is invisible to Sweep regardless of
// deadline, and ResolveAndEmit's transition guard would refuse it again even
// if it weren't.
func TestSweepDoesNotReSweepATerminalRequest(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	p := NewProcessor(l, ctx, db)
	characterId := seedCharacter(t, db, "Sierra", world.Id(0))

	assetId := uint32(9005)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Sierratwo", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	if _, _, err := p.ResolveAndEmit(m.Id(), StatusCancelled, "operator_cancel"); err != nil {
		t.Fatalf("ResolveAndEmit cancel: %v", err)
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("expected exactly one refund from the cancel")
	}

	if err := db.Model(&entity{}).Where("id = ?", m.Id()).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}

	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	got, err := p.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != StatusCancelled {
		t.Fatalf("got status %s, want CANCELLED (unchanged)", got.Status())
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("sweep refunded an already-terminal request a second time")
	}
}

// TestSweepIsTenantScoped proves a Sweep run under tenant A's context never
// touches tenant B's expired rows -- both are backdated past due, but only
// tenant A's context is swept.
func TestSweepIsTenantScoped(t *testing.T) {
	db := newProcessorTestDB(t)

	tenantA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create A: %v", err)
	}
	tenantB, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create B: %v", err)
	}
	ctxA := tenant.WithContext(context.Background(), tenantA)
	ctxB := tenant.WithContext(context.Background(), tenantB)

	pA := NewProcessor(testLogger(t), ctxA, db)
	pB := NewProcessor(testLogger(t), ctxB, db)

	characterIdA := seedCharacterInContext(t, ctxA, db, "Tango", world.Id(0))
	characterIdB := seedCharacterInContext(t, ctxB, db, "Uniform", world.Id(0))

	assetIdA, assetIdB := uint32(9006), uint32(9007)
	mA, err := pA.CreateAndEmit(uuid.New(), characterIdA, TypeNameChange, "Tangotwo", world.Id(0), &assetIdA)
	if err != nil {
		t.Fatalf("CreateAndEmit A: %v", err)
	}
	mB, err := pB.CreateAndEmit(uuid.New(), characterIdB, TypeNameChange, "Uniformtwo", world.Id(0), &assetIdB)
	if err != nil {
		t.Fatalf("CreateAndEmit B: %v", err)
	}

	past := time.Now().Add(-time.Minute)
	if err := db.Model(&entity{}).Where("id = ?", mA.Id()).Update("expires_at", past).Error; err != nil {
		t.Fatalf("backdate A: %v", err)
	}
	if err := db.Model(&entity{}).Where("id = ?", mB.Id()).Update("expires_at", past).Error; err != nil {
		t.Fatalf("backdate B: %v", err)
	}

	if err := pA.Sweep(time.Now()); err != nil {
		t.Fatalf("Sweep A: %v", err)
	}

	gotA, err := pA.GetById(mA.Id())
	if err != nil {
		t.Fatalf("GetById A: %v", err)
	}
	if gotA.Status() != StatusExpired {
		t.Fatalf("got tenant A status %s, want EXPIRED", gotA.Status())
	}

	gotB, err := pB.GetById(mB.Id())
	if err != nil {
		t.Fatalf("GetById B: %v", err)
	}
	if gotB.Status() != StatusPending {
		t.Fatalf("tenant B's row was swept by tenant A's Sweep call: got status %s, want PENDING", gotB.Status())
	}
}

// TestExpiryRunSweepsEachTenantUnderItsOwnContext exercises the ticker
// itself: two tenants each have one expired PENDING row. Run enumerates the
// distinct expired tenant ids directly from the table, resolves each via the
// injected stub tenantFetcher (standing in for atlas-tenants), and sweeps
// each tenant under its own tenant.WithContext -- proving both rows resolve
// in a single tick and neither leaks the other's Model.
func TestExpiryRunSweepsEachTenantUnderItsOwnContext(t *testing.T) {
	db := newProcessorTestDB(t)

	tenantA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create A: %v", err)
	}
	tenantB, err := tenant.Create(uuid.New(), "GMS", 87, 1)
	if err != nil {
		t.Fatalf("tenant.Create B: %v", err)
	}
	ctxA := tenant.WithContext(context.Background(), tenantA)
	ctxB := tenant.WithContext(context.Background(), tenantB)

	pA := NewProcessor(testLogger(t), ctxA, db)
	pB := NewProcessor(testLogger(t), ctxB, db)

	characterIdA := seedCharacterInContext(t, ctxA, db, "Victor", world.Id(0))
	characterIdB := seedCharacterInContext(t, ctxB, db, "Whiskey", world.Id(0))

	assetIdA, assetIdB := uint32(9008), uint32(9009)
	mA, err := pA.CreateAndEmit(uuid.New(), characterIdA, TypeNameChange, "Victortwo", world.Id(0), &assetIdA)
	if err != nil {
		t.Fatalf("CreateAndEmit A: %v", err)
	}
	mB, err := pB.CreateAndEmit(uuid.New(), characterIdB, TypeNameChange, "Whiskeytwo", world.Id(0), &assetIdB)
	if err != nil {
		t.Fatalf("CreateAndEmit B: %v", err)
	}

	past := time.Now().Add(-time.Minute)
	if err := db.Model(&entity{}).Where("id = ?", mA.Id()).Update("expires_at", past).Error; err != nil {
		t.Fatalf("backdate A: %v", err)
	}
	if err := db.Model(&entity{}).Where("id = ?", mB.Id()).Update("expires_at", past).Error; err != nil {
		t.Fatalf("backdate B: %v", err)
	}

	models := map[uuid.UUID]tenant.Model{
		tenantA.Id(): tenantA,
		tenantB.Id(): tenantB,
	}
	e := newExpiryWithFetcher(testLogger(t), db, time.Minute, tenantStub(models))
	e.Run()

	gotA, err := pA.GetById(mA.Id())
	if err != nil {
		t.Fatalf("GetById A: %v", err)
	}
	if gotA.Status() != StatusExpired {
		t.Fatalf("got tenant A status %s, want EXPIRED", gotA.Status())
	}
	gotB, err := pB.GetById(mB.Id())
	if err != nil {
		t.Fatalf("GetById B: %v", err)
	}
	if gotB.Status() != StatusExpired {
		t.Fatalf("got tenant B status %s, want EXPIRED", gotB.Status())
	}
}

// tenantStub builds a tenantFetcher backed by a fixed id->Model map, standing
// in for a live atlas-tenants call in Run()-level tests.
func tenantStub(models map[uuid.UUID]tenant.Model) tenantFetcher {
	return func(_ logrus.FieldLogger, _ context.Context, tenantId uuid.UUID) (tenant.Model, error) {
		return models[tenantId], nil
	}
}
