package scheduling

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, MigrateTable)
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(uuid.New())
}

var at = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// FR-B4/FR-S8 guard 1: a redelivered Kafka message must be a no-op insert, not
// a second work row and not an error.
func TestScheduleDedupesOnKey(t *testing.T) {
	defId := uuid.New()
	db := newTestDB(t)
	a := NewAdministrator(testLogger(t), testCtx(t), db)

	m, _ := NewBuilder(defId, WorkTypeTriggerEvaluation).
		SetExecuteAt(at).
		SetDedupeKey("balrog:d1:v1:1:4").
		Build()

	first, created, err := a.Schedule(m)
	if err != nil || !created {
		t.Fatalf("first schedule: created=%v err=%v", created, err)
	}
	second, created, err := a.Schedule(m)
	if err != nil {
		t.Fatalf("redelivery must not error: %v", err)
	}
	if created {
		t.Fatalf("redelivery created a second row")
	}
	if second.Id() != first.Id() {
		t.Fatalf("redelivery returned a different row")
	}

	var count int64
	db.Model(&Entity{}).Count(&count)
	if count != 1 {
		t.Fatalf("dedupe key was inserted %d times, want 1", count)
	}
}

// The dedupe index is PARTIAL on (PENDING, PROCESSING), so a cancelled or
// failed row does not block a legitimate retry of the same logical work.
func TestDedupeDoesNotBlockAfterCancellation(t *testing.T) {
	defId := uuid.New()
	db := newTestDB(t)
	a := NewAdministrator(testLogger(t), testCtx(t), db)
	m, _ := NewBuilder(defId, WorkTypeTriggerEvaluation).SetExecuteAt(at).SetDedupeKey("k").Build()

	first, _, _ := a.Schedule(m)
	if _, err := a.SetState(first.Id(), StateCancelled, ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if _, created, err := a.Schedule(m); err != nil || !created {
		t.Fatalf("re-schedule after cancel: created=%v err=%v", created, err)
	}

	var count int64
	db.Model(&Entity{}).Where("dedupe_key = ?", "k").Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows (cancelled + retried), got %d", count)
	}
}

// An empty dedupe key opts out — an OCCURRENCE_TRANSITION scheduled per
// occurrence needs no cross-message dedup.
func TestEmptyDedupeKeyAllowsMany(t *testing.T) {
	defId := uuid.New()
	db := newTestDB(t)
	a := NewAdministrator(testLogger(t), testCtx(t), db)

	m, _ := NewBuilder(defId, WorkTypeOccurrenceTransition).SetExecuteAt(at).Build()

	first, created, err := a.Schedule(m)
	if err != nil || !created {
		t.Fatalf("first schedule: created=%v err=%v", created, err)
	}
	second, created, err := a.Schedule(m)
	if err != nil || !created {
		t.Fatalf("second schedule: created=%v err=%v", created, err)
	}
	if first.Id() == second.Id() {
		t.Fatalf("two schedules with an empty dedupe key produced the same row")
	}

	var count int64
	db.Model(&Entity{}).Where("event_definition_id = ?", defId).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}

// FR-S10: a definition's pending work can be cancelled — e.g. an Anniversary
// definition whose start time is edited before it fires. Only PENDING rows are
// affected; a row already PROCESSING belongs to a claimer.
func TestCancelPendingForDefinitionLeavesClaimedWorkAlone(t *testing.T) {
	defId := uuid.New()
	db := newTestDB(t)
	ctx := testCtx(t)
	a := NewAdministrator(testLogger(t), ctx, db)

	pending, _, err := a.Schedule(mustBuild(t, defId, "pending"))
	if err != nil {
		t.Fatalf("schedule pending: %v", err)
	}
	processing, _, err := a.Schedule(mustBuild(t, defId, "processing"))
	if err != nil {
		t.Fatalf("schedule processing: %v", err)
	}
	if _, err := a.SetState(processing.Id(), StateProcessing, ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	cancelled, err := CancelPendingForDefinition(db.WithContext(ctx))(defId)
	if err != nil {
		t.Fatalf("CancelPendingForDefinition: %v", err)
	}
	if cancelled != 1 {
		t.Fatalf("cancelled = %d, want 1", cancelled)
	}

	var pendingEntity Entity
	if err := db.Where("id = ?", pending.Id()).First(&pendingEntity).Error; err != nil {
		t.Fatalf("reload pending: %v", err)
	}
	if pendingEntity.State != StateCancelled {
		t.Fatalf("pending row state = %s, want %s", pendingEntity.State, StateCancelled)
	}

	var processingEntity Entity
	if err := db.Where("id = ?", processing.Id()).First(&processingEntity).Error; err != nil {
		t.Fatalf("reload processing: %v", err)
	}
	if processingEntity.State != StateProcessing {
		t.Fatalf("processing row state = %s, want left alone at %s", processingEntity.State, StateProcessing)
	}
}

func mustBuild(t *testing.T, defId uuid.UUID, dedupeKey string) Model {
	t.Helper()
	m, err := NewBuilder(defId, WorkTypeTriggerEvaluation).SetExecuteAt(at).SetDedupeKey(dedupeKey).Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return m
}

// SetState on a missing id surfaces gorm.ErrRecordNotFound rather than
// silently no-oping.
func TestSetStateReturnsRecordNotFoundForMissingId(t *testing.T) {
	db := newTestDB(t)
	a := NewAdministrator(testLogger(t), testCtx(t), db)

	if _, err := a.SetState(uuid.New(), StateCompleted, ""); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("SetState on a missing id = %v, want gorm.ErrRecordNotFound", err)
	}
}

// TestOnConflictDoNothingSignalsViaRowsAffected proves the mechanism
// Schedule's insert-race fallback depends on: once a unique index exists on
// dedupe_key (added by Task 19 as ux_sew_dedupe), `Clauses(clause.OnConflict{
// DoNothing: true}).Create(...)` reports a losing insert via
// RowsAffected == 0 — not a driver error — so a caller can re-read the
// winning row on the SAME transaction handle without ever having issued a
// failed SQL statement.
//
// It cannot prove Postgres's transaction-abort behavior: SQLite has no such
// semantics, so this test would also pass if Schedule instead caught a
// failed INSERT's error and re-queried afterward — the exact shape this fix
// replaced. What it DOES prove is that the ON CONFLICT clause and the
// RowsAffected==0 signal it depends on are wired correctly here, using the
// same idiom event/occurrence's createFromSeed already relies on
// (event/occurrence/administrator.go) for its own concurrency-key race.
func TestOnConflictDoNothingSignalsViaRowsAffected(t *testing.T) {
	db := newTestDB(t)
	ctx := testCtx(t)
	scoped := db.WithContext(ctx)

	if err := db.Exec(`CREATE UNIQUE INDEX ux_sew_dedupe_test ON scheduled_event_work (dedupe_key) WHERE state IN ('PENDING','PROCESSING')`).Error; err != nil {
		t.Fatalf("create test index: %v", err)
	}

	defId := uuid.New()
	winner := mustBuild(t, defId, "race-key")
	winnerEntity, err := ToEntity(winner, testTenantId(t, ctx))
	if err != nil {
		t.Fatalf("ToEntity(winner): %v", err)
	}
	if err := scoped.Create(&winnerEntity).Error; err != nil {
		t.Fatalf("insert winner: %v", err)
	}

	loser := mustBuild(t, defId, "race-key")
	loserEntity, err := ToEntity(loser, testTenantId(t, ctx))
	if err != nil {
		t.Fatalf("ToEntity(loser): %v", err)
	}

	res := scoped.Clauses(clause.OnConflict{DoNothing: true}).Create(&loserEntity)
	if res.Error != nil {
		t.Fatalf("ON CONFLICT DO NOTHING surfaced a driver error instead of RowsAffected==0: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Fatalf("RowsAffected = %d, want 0 (the losing insert must be silently absorbed)", res.RowsAffected)
	}

	var count int64
	scoped.Model(&Entity{}).Where("dedupe_key = ?", "race-key").Count(&count)
	if count != 1 {
		t.Fatalf("row count = %d, want 1 (loser must not have been inserted)", count)
	}
}

func testTenantId(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	id, err := tenant.FromContext(ctx)()
	if err != nil {
		t.Fatalf("no tenant in context: %v", err)
	}
	return id.Id()
}
