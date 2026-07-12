package holding_test

import (
	"atlas-mts/holding"
	"atlas-mts/saga"
	"atlas-mts/test"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// captureEmitter records the saga handed to it instead of producing to Kafka, so
// the take-home flow can be asserted without a live broker. Mirrors the listing
// package's captureEmitter (Task 4.1).
type captureEmitter struct {
	saga   saga.Saga
	called bool
}

func (e *captureEmitter) Create(s saga.Saga) error {
	e.saga = s
	e.called = true
	return nil
}

// newTakeHomeProcessor builds a holding processor wired to a capturing saga
// emitter, mirroring listing.newListProcessor (Task 4.1).
func newTakeHomeProcessor(t *testing.T) (holding.Processor, *captureEmitter, *gorm.DB, func()) {
	t.Helper()
	logger := logrus.New()
	db := test.SetupTestDB(t, holding.Migration)
	ctx := test.CreateTestContext()
	emitter := &captureEmitter{}
	p := holding.NewProcessor(logger, ctx, db, holding.WithSagaEmitter(emitter))
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset holdings: %v", err)
	}
	cleanup := func() { test.CleanupTestDB(t, db) }
	return p, emitter, db, cleanup
}

func seedTakeHomeHolding(t *testing.T, p holding.Processor, worldId world.Id, ownerId uint32) holding.Model {
	t.Helper()
	m, err := holding.NewBuilder(test.TestTenantId, worldId, ownerId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("create holding: %v", err)
	}
	return created
}

// TestTakeHomeBuildsWithdrawFromMtsSaga asserts a take-home builds and emits a
// WithdrawFromMts saga carrying the holding id, owner character id, world, and
// inventory type, with an explicit step-count-scaled timeout (N=2: the
// WithdrawFromMts composite expands to release_from_mts_holding +
// accept_to_character). It also asserts the holding row is NOT directly
// soft-deleted by take-home initiation — the saga's ReleaseFromMtsHolding custody
// command does that (idempotently on replay; see
// TestReleaseFromMtsHolding_SoftDeletesAndIsIdempotent in the custody consumer).
func TestTakeHomeBuildsWithdrawFromMtsSaga(t *testing.T) {
	p, emitter, db, cleanup := newTakeHomeProcessor(t)
	defer cleanup()

	created := seedTakeHomeHolding(t, p, 0, 100)

	const inventoryType = byte(1)
	const slot = int16(3)
	txnId, err := p.TakeHome(created.Id().String(), 100, 0, inventoryType, slot)
	if err != nil {
		t.Fatalf("TakeHome: %v", err)
	}
	if txnId == uuid.Nil {
		t.Fatal("TakeHome did not allocate a transaction id")
	}
	if !emitter.called {
		t.Fatal("expected a saga to be emitted for take-home")
	}

	sg := emitter.saga
	if sg.SagaType != saga.MtsOperation {
		t.Errorf("saga type = %q, want %q", sg.SagaType, saga.MtsOperation)
	}
	if sg.TransactionId != txnId {
		t.Errorf("saga transactionId = %s, want %s", sg.TransactionId, txnId)
	}
	if len(sg.Steps) != 1 {
		t.Fatalf("expected 1 step (WithdrawFromMts composite), got %d", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.WithdrawFromMts {
		t.Errorf("step[0] action = %q, want %q", sg.Steps[0].Action, saga.WithdrawFromMts)
	}
	wp, ok := sg.Steps[0].Payload.(sharedsaga.WithdrawFromMtsPayload)
	if !ok {
		t.Fatalf("step[0] payload type = %T, want WithdrawFromMtsPayload", sg.Steps[0].Payload)
	}
	if wp.HoldingId != created.Id() {
		t.Errorf("WithdrawFromMts holdingId = %s, want %s", wp.HoldingId, created.Id())
	}
	if wp.CharacterId != 100 {
		t.Errorf("WithdrawFromMts characterId = %d, want 100", wp.CharacterId)
	}
	if wp.WorldId != 0 {
		t.Errorf("WithdrawFromMts worldId = %d, want 0", wp.WorldId)
	}
	if wp.InventoryType != inventoryType {
		t.Errorf("WithdrawFromMts inventoryType = %d, want %d", wp.InventoryType, inventoryType)
	}
	if wp.TransactionId != txnId {
		t.Errorf("WithdrawFromMts transactionId = %s, want %s", wp.TransactionId, txnId)
	}

	// Timeout must be set (never default) and scaled for the N=2 expansion of the
	// WithdrawFromMts composite (release_from_mts_holding + accept_to_character).
	if sg.Timeout <= 0 {
		t.Errorf("saga timeout = %d, want a positive explicit timeout", sg.Timeout)
	}
	got := time.Duration(sg.Timeout) * time.Millisecond
	// base 10s + perStep 1s * 2 = 12s
	if got < 12*time.Second {
		t.Errorf("saga timeout = %s, want at least the N=2 scaled budget (>= 12s)", got)
	}

	// The holding row must NOT be soft-deleted directly by take-home initiation —
	// the saga's ReleaseFromMtsHolding custody command does that. The row is still
	// present and visible to GetByOwner here.
	rows, err := p.GetByOwner(0, 100)
	if err != nil {
		t.Fatalf("GetByOwner after take-home: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("GetByOwner after take-home returned %d rows, want 1 (the row is released by the saga, not by initiation)", len(rows))
	}
	_ = db
}
