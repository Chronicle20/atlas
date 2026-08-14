package pending_change

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

var testTenantModel = func() tenant.Model {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		panic(err)
	}
	return tm
}()

func testContext(t *testing.T) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), testTenantModel)
}

// newProcessorTestDB brings up the disposable Postgres of entity_test.go and
// migrates everything the processor touches: the pending-change table (with its
// two partial unique indexes), the characters table the apply path writes, and
// the outbox table every emission lands in.
func newProcessorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newTestDB(t)
	for _, m := range []func(*gorm.DB) error{Migration, character.Migration, outbox.Migration} {
		if err := m(db); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}
	return db
}

// seedCharacter creates a real character row through the character processor so
// the pending-change processor's GetById / CheckNameValidity see exactly what
// production sees.
func seedCharacter(t *testing.T, db *gorm.DB, name string, worldId world.Id) uint32 {
	t.Helper()
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(worldId).
		SetName(name).
		SetLevel(1).
		SetExperience(0).
		Build()
	c, err := character.NewProcessor(testLogger(t), testContext(t), db).
		Create(message.NewBuffer())(uuid.New(), input, 0)
	if err != nil {
		t.Fatalf("seed character %s: %v", name, err)
	}
	return c.Id()
}

// countOutboxMessagesMatching counts outbox rows whose serialized body contains
// substr. Reading the outbox directly (rather than a message.Buffer) is the
// point: it is the only place an emission survives the transaction, so counting
// here proves what actually reaches Kafka, across every invocation in the test.
func countOutboxMessagesMatching(t *testing.T, db *gorm.DB, substr string) int {
	t.Helper()
	var n int64
	err := db.Model(&outbox.Entity{}).
		Where("encode(message_value, 'escape') LIKE ?", "%"+substr+"%").
		Count(&n).Error
	if err != nil {
		t.Fatalf("count outbox messages matching %q: %v", substr, err)
	}
	return int(n)
}

// A redelivered cancel must not mint a second coupon. The guard is the
// transition's RowsAffected, not a handler-level dedupe: only the transition
// that actually moves status out of PENDING emits the refund.
//
// The count spans ALL THREE deliveries, not just the redeliveries. The failure
// mode this guards is a duplicate produced by the first and second calls
// together, which a "second call errored" assertion would not catch.
func TestRedeliveredCancelRefundsExactlyOnce(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)

	characterId := seedCharacter(t, db, "Golf", world.Id(0))

	assetId := uint32(9001)
	p := NewProcessor(l, ctx, db)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Hotel", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	awardsBefore := countOutboxMessagesMatching(t, db, "award_asset")

	for i := 0; i < 3; i++ {
		if _, moved, err := p.ResolveAndEmit(m.Id(), StatusCancelled, "operator_cancelled"); err != nil {
			t.Fatalf("delivery %d: %v", i, err)
		} else if moved != (i == 0) {
			t.Fatalf("delivery %d: moved = %v, want %v", i, moved, i == 0)
		}
	}

	if got := countOutboxMessagesMatching(t, db, "award_asset") - awardsBefore; got != 1 {
		t.Fatalf("expected exactly 1 refund emission across 3 deliveries, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED"); got != 1 {
		t.Fatalf("expected exactly 1 resolved notification, got %d", got)
	}
	// The coupon was consumed exactly once too, at acceptance.
	if got := countOutboxMessagesMatching(t, db, "destroy_asset"); got != 1 {
		t.Fatalf("expected exactly 1 consumption command, got %d", got)
	}
}

// A purchase-path record carries no asset id; resolution must still notify, and
// must not emit a refund with a zero asset.
func TestPurchasePathResolutionEmitsNoAssetRefund(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "India", world.Id(0))

	p := NewProcessor(testLogger(t), testContext(t), db).withTransferEligibilityGates(passingGateDeps())
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeWorldTransfer, "", world.Id(2), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if _, moved, err := p.ResolveAndEmit(m.Id(), StatusExpired, "expired"); err != nil || !moved {
		t.Fatalf("ResolveAndEmit: moved=%v err=%v", moved, err)
	}
	if got := countOutboxMessagesMatching(t, db, "award_asset"); got != 0 {
		t.Fatalf("expected no asset refund on the purchase path, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "destroy_asset"); got != 0 {
		t.Fatalf("expected no consumption command on the purchase path, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED"); got != 1 {
		t.Fatalf("expected 1 resolved notification, got %d", got)
	}
}

// An APPLIED exit is the one terminal status that must NOT refund — the player
// got what they paid for.
func TestAppliedResolutionEmitsNoRefund(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "Juliett", world.Id(0))

	assetId := uint32(9005)
	p := NewProcessor(testLogger(t), testContext(t), db)
	m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Kilo", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if _, moved, err := p.ResolveAndEmit(m.Id(), StatusApplied, ""); err != nil || !moved {
		t.Fatalf("ResolveAndEmit: moved=%v err=%v", moved, err)
	}
	if got := countOutboxMessagesMatching(t, db, "award_asset"); got != 0 {
		t.Fatalf("expected no refund on the APPLIED path, got %d", got)
	}
}
