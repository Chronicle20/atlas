package scheduling

// poller_test.go — NO build tag; runs in the default `go test ./...` gate.
//
// SQLite silently drops the clause.Locking clause ClaimBatch issues, instead
// of erroring (gorm.io/driver/sqlite@v1.6.0/sqlite.go:124-129), so every test
// in this file proves the design §5.2 outcome state machine — claim
// eligibility, lease reclaim, retry/backoff/fail transitions, dispatch
// wiring, unregistered-type handling — and NOTHING about SKIP LOCKED's
// row-isolation guarantee under concurrent claimers. That guarantee is
// proven separately, against real Postgres, by
// poller_integration_test.go's TestTwoInstancesNeverExecuteTheSameRow
// (controller ruling, task-18 brief).

import (
	"atlas-events/event/definition"
	"atlas-events/event/occurrence"
	"atlas-events/event/registry"
	"atlas-events/event/transition"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func newPollerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t,
		MigrateTable, definition.MigrateTable, occurrence.MigrateTable, transition.MigrateTable)
}

// insertDefinition inserts an event_definition row of theType, returning its id.
func insertDefinition(t *testing.T, db *gorm.DB, ctx context.Context, theType string, enabled bool) uuid.UUID {
	t.Helper()
	tm := testTenant(t, ctx)
	e := definition.Entity{
		ID: uuid.New(), TenantID: tm.Id(), Type: theType, Name: theType,
		Enabled: enabled, Configuration: "{}",
	}
	if err := db.WithContext(ctx).Create(&e).Error; err != nil {
		t.Fatalf("insert definition: %v", err)
	}
	return e.ID
}

// insertWork inserts a scheduled_event_work row of type TRIGGER_EVALUATION
// against a fresh enabled definition of theType, in the given state due at
// executeAt. Returns the work row id.
func insertWork(t *testing.T, db *gorm.DB, ctx context.Context, theType string, state string, executeAt time.Time) uuid.UUID {
	t.Helper()
	defId := insertDefinition(t, db, ctx, theType, true)
	m, err := NewBuilder(defId, WorkTypeTriggerEvaluation).
		SetExecuteAt(executeAt).
		SetState(state).
		Build()
	if err != nil {
		t.Fatalf("build work: %v", err)
	}
	entity, err := ToEntity(m, testTenant(t, ctx))
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if err := db.WithContext(ctx).Create(&entity).Error; err != nil {
		t.Fatalf("insert work: %v", err)
	}
	return entity.ID
}

// insertClaimedWork inserts a PROCESSING row claimed by claimedBy at claimedAt.
func insertClaimedWork(t *testing.T, db *gorm.DB, ctx context.Context, claimedBy string, claimedAt time.Time) uuid.UUID {
	t.Helper()
	id := insertWork(t, db, ctx, "CLAIMED_TYPE", StateProcessing, time.Now())
	if err := db.WithContext(ctx).Model(&Entity{}).Where("id = ?", id).
		Updates(map[string]any{"claimed_by": claimedBy, "claimed_at": claimedAt}).Error; err != nil {
		t.Fatalf("mark claimed: %v", err)
	}
	return id
}

func readWork(t *testing.T, db *gorm.DB, id uuid.UUID) Entity {
	t.Helper()
	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read work: %v", err)
	}
	return e
}

// FR-S5: work whose executeAt passed while the service was down runs on
// recovery — late, not lost.
func TestOverdueWorkIsClaimedOnFirstPoll(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	insertWork(t, db, ctx, "OVERDUE_TYPE", StatePending, time.Now().Add(-2*time.Hour))

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed %d overdue rows, want 1", len(claimed))
	}
}

// Work not yet due is not claimed.
func TestFutureWorkIsNotClaimed(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	insertWork(t, db, ctx, "FUTURE_TYPE", StatePending, time.Now().Add(2*time.Hour))

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("claimed %d future rows, want 0", len(claimed))
	}
}

// FR-S7: a row left PROCESSING by a dead replica returns to PENDING after the
// lease, with attempts incremented.
func TestLeaseReclaimReturnsOrphanedWork(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	id := insertClaimedWork(t, db, ctx, "dead-replica", time.Now().Add(-10*time.Minute))

	p := NewProcessor(testLogger(t), ctx, db)
	n, err := p.Reclaim(5 * time.Minute)
	if err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	if n != 1 {
		t.Fatalf("reclaimed %d, want 1", n)
	}
	got := readWork(t, db, id)
	if got.State != StatePending || got.Attempts != 1 {
		t.Fatalf("state=%s attempts=%d, want PENDING/1", got.State, got.Attempts)
	}
}

// FR-S9: repeated failure lands in FAILED with lastError retained, and a
// FAILED row never blocks a PENDING sibling.
func TestRepeatedFailureLandsInFailedWithoutBlockingSiblings(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	failing := insertWork(t, db, ctx, "FAILING_TYPE", StatePending, time.Now().Add(-time.Minute))
	sibling := insertWork(t, db, ctx, "SIBLING_TYPE", StatePending, time.Now().Add(-time.Minute))

	p := NewProcessorWithExecutor(testLogger(t), ctx, db, func(Model) error {
		return errors.New("boom")
	})
	p.SetMaxAttempts(2)

	for i := 0; i < 3; i++ {
		claimed, _ := p.ClaimBatch("a", 10)
		for _, m := range claimed {
			_ = p.ExecuteOne(m)
		}
		_, _ = p.Reclaim(0)
	}

	f := readWork(t, db, failing)
	if f.State != StateFailed || f.LastError == "" {
		t.Fatalf("failing row: state=%s lastError=%q", f.State, f.LastError)
	}
	s := readWork(t, db, sibling)
	if s.State == StatePending {
		t.Fatalf("sibling was never processed — a FAILED row blocked the queue")
	}
}

// A definition whose type has no registered handler makes its work FAIL
// loudly with a named reason, rather than silently completing (design §3.2).
func TestWorkForAnUnregisteredTypeFails(t *testing.T) {
	registry.ResetForTest()
	defer registry.ResetForTest()

	db := newPollerTestDB(t)
	ctx := testCtx(t)
	id := insertWork(t, db, ctx, "TOTALLY_UNKNOWN_TYPE", StatePending, time.Now().Add(-time.Minute))

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	// ExecuteOne's return value reports whether the row's outcome WRITE
	// succeeded, not whether the handler itself succeeded — the handler
	// failure is recorded on the row (asserted below), same as every other
	// outcome-table test in this file.
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, id)
	if got.State != StateFailed {
		t.Fatalf("state = %s, want FAILED", got.State)
	}
	if !strings.Contains(got.LastError, "no handler for type") {
		t.Fatalf("lastError = %q", got.LastError)
	}
}

// --- design §5.2 outcome table, row by row ---

// Row: handler returned normally -> COMPLETED.
func TestOutcomeHandlerSuccessCompletes(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	id := insertWork(t, db, ctx, "OK_TYPE", StatePending, time.Now().Add(-time.Minute))

	p := NewProcessorWithExecutor(testLogger(t), ctx, db, func(Model) error { return nil })
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, id)
	if got.State != StateCompleted {
		t.Fatalf("state = %s, want COMPLETED", got.State)
	}
}

// Row: handler errored, attempts < max -> PENDING, execute_at bumped by
// backoff, last_error set.
func TestOutcomeHandlerErrorBelowMaxAttemptsRetries(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	id := insertWork(t, db, ctx, "RETRY_TYPE", StatePending, time.Now().Add(-time.Minute))

	p := NewProcessorWithExecutor(testLogger(t), ctx, db, func(Model) error { return errors.New("transient") })
	p.SetMaxAttempts(5)
	p.SetBackoff(time.Hour)

	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, id)
	if got.State != StatePending {
		t.Fatalf("state = %s, want PENDING", got.State)
	}
	if got.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", got.Attempts)
	}
	if got.LastError == "" {
		t.Fatalf("lastError not set")
	}
	if !got.ExecuteAt.After(time.Now().Add(30 * time.Minute)) {
		t.Fatalf("executeAt = %v, want bumped by backoff", got.ExecuteAt)
	}
}

// Row: handler errored, attempts >= max -> FAILED, last_error retained.
func TestOutcomeHandlerErrorAtMaxAttemptsFails(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	id := insertWork(t, db, ctx, "FAIL_TYPE", StatePending, time.Now().Add(-time.Minute))

	p := NewProcessorWithExecutor(testLogger(t), ctx, db, func(Model) error { return errors.New("permanent") })
	p.SetMaxAttempts(1)

	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, id)
	if got.State != StateFailed {
		t.Fatalf("state = %s, want FAILED", got.State)
	}
	if got.LastError == "" {
		t.Fatalf("lastError not retained")
	}
}

// Row: definition disabled -> COMPLETED, no occurrence.
func TestOutcomeDisabledDefinitionCompletesWithNoOccurrence(t *testing.T) {
	db := newPollerTestDB(t)
	ctx := testCtx(t)
	defId := insertDefinition(t, db, ctx, "DISABLED_TYPE", false)
	m, err := NewBuilder(defId, WorkTypeTriggerEvaluation).SetExecuteAt(time.Now().Add(-time.Minute)).Build()
	if err != nil {
		t.Fatalf("build work: %v", err)
	}
	entity, err := ToEntity(m, testTenant(t, ctx))
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if err := db.WithContext(ctx).Create(&entity).Error; err != nil {
		t.Fatalf("insert work: %v", err)
	}

	p := NewProcessor(testLogger(t), ctx, db) // real dispatch, no executor override
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, entity.ID)
	if got.State != StateCompleted {
		t.Fatalf("state = %s, want COMPLETED", got.State)
	}
	var count int64
	db.Model(&occurrence.Entity{}).Where("event_definition_id = ?", defId).Count(&count)
	if count != 0 {
		t.Fatalf("occurrence count = %d, want 0", count)
	}
}

// --- real dispatch wiring (registry.Get -> Evaluate/Advance/Start), covering
// what the outcome-table tests above deliberately bypass via executor
// override ---

// fakeHandler is a minimal registry.Handler test double.
type fakeHandler struct {
	theType  string
	evaluate func(ctx context.Context, d registry.Definition, w registry.Work) (*registry.Seed, error)
	start    func(ctx context.Context, o registry.Occurrence) (registry.Progress, error)
	advance  func(ctx context.Context, o registry.Occurrence, w registry.Work) (registry.Progress, error)
}

func (h fakeHandler) Type() string                                { return h.theType }
func (h fakeHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (h fakeHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}

func (h fakeHandler) ConcurrencyKeyIsConstant() bool { return false }
func (h fakeHandler) Evaluate(ctx context.Context, d registry.Definition, w registry.Work) (*registry.Seed, error) {
	if h.evaluate != nil {
		return h.evaluate(ctx, d, w)
	}
	return nil, nil
}

func (h fakeHandler) Start(ctx context.Context, o registry.Occurrence) (registry.Progress, error) {
	if h.start != nil {
		return h.start(ctx, o)
	}
	return registry.Progress{}, nil
}

func (h fakeHandler) Advance(ctx context.Context, o registry.Occurrence, w registry.Work) (registry.Progress, error) {
	if h.advance != nil {
		return h.advance(ctx, o, w)
	}
	return registry.Progress{}, nil
}

var _ registry.Handler = fakeHandler{}

// Evaluate returning a Seed drives CreateFromSeed + Start, and the work row
// completes.
func TestExecuteOneCreatesOccurrenceFromSeedAndStarts(t *testing.T) {
	registry.ResetForTest()
	defer registry.ResetForTest()
	const theType = "DISPATCH_TYPE"
	registry.Register(fakeHandler{
		theType: theType,
		evaluate: func(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
			return &registry.Seed{Stage: "stage1", ConcurrencyKey: "ck-1"}, nil
		},
		start: func(context.Context, registry.Occurrence) (registry.Progress, error) {
			return registry.Progress{Stage: "stage2"}, nil
		},
	})

	db := newPollerTestDB(t)
	ctx := testCtx(t)
	workId := insertWork(t, db, ctx, theType, StatePending, time.Now().Add(-time.Minute))

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, workId)
	if got.State != StateCompleted {
		t.Fatalf("work state = %s, want COMPLETED", got.State)
	}

	var occCount int64
	db.Model(&occurrence.Entity{}).Where("concurrency_key = ?", "ck-1").Count(&occCount)
	if occCount != 1 {
		t.Fatalf("occurrence count = %d, want 1", occCount)
	}
}

// An OCCURRENCE_TRANSITION row drives Advance against an existing occurrence.
func TestExecuteOneAdvancesExistingOccurrence(t *testing.T) {
	registry.ResetForTest()
	defer registry.ResetForTest()
	const theType = "ADVANCE_TYPE"
	registry.Register(fakeHandler{
		theType: theType,
		advance: func(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
			return registry.Progress{Stage: "advanced"}, nil
		},
	})

	db := newPollerTestDB(t)
	ctx := testCtx(t)
	defId := insertDefinition(t, db, ctx, theType, true)

	om, err := occurrence.NewBuilder(defId, theType).SetStage("initial").Build()
	if err != nil {
		t.Fatalf("build occurrence: %v", err)
	}
	tm := testTenant(t, ctx)
	oe, err := occurrence.ToEntity(om, tm.Id())
	if err != nil {
		t.Fatalf("ToEntity occurrence: %v", err)
	}
	if err := db.WithContext(ctx).Create(&oe).Error; err != nil {
		t.Fatalf("insert occurrence: %v", err)
	}

	m, err := NewBuilder(defId, WorkTypeOccurrenceTransition).
		SetOccurrenceId(oe.ID).
		SetExecuteAt(time.Now().Add(-time.Minute)).
		Build()
	if err != nil {
		t.Fatalf("build work: %v", err)
	}
	we, err := ToEntity(m, testTenant(t, ctx))
	if err != nil {
		t.Fatalf("ToEntity work: %v", err)
	}
	if err := db.WithContext(ctx).Create(&we).Error; err != nil {
		t.Fatalf("insert work: %v", err)
	}

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	gotWork := readWork(t, db, we.ID)
	if gotWork.State != StateCompleted {
		t.Fatalf("work state = %s, want COMPLETED", gotWork.State)
	}

	var gotOcc occurrence.Entity
	if err := db.Where("id = ?", oe.ID).First(&gotOcc).Error; err != nil {
		t.Fatalf("read occurrence: %v", err)
	}
	if gotOcc.Stage != "advanced" {
		t.Fatalf("occurrence stage = %s, want advanced", gotOcc.Stage)
	}
}

// A concurrency-key collision on CreateFromSeed is treated as success — the
// other racer already created the occurrence.
func TestExecuteOneTreatsConcurrencyKeyTakenAsSuccess(t *testing.T) {
	registry.ResetForTest()
	defer registry.ResetForTest()
	const theType = "RACE_TYPE"
	registry.Register(fakeHandler{
		theType: theType,
		evaluate: func(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
			return &registry.Seed{Stage: "stage1", ConcurrencyKey: "ck-race"}, nil
		},
	})

	db := newPollerTestDB(t)
	ctx := testCtx(t)
	defId := insertDefinition(t, db, ctx, theType, true)

	// Pre-seed the winning occurrence directly, holding the concurrency key.
	om, err := occurrence.NewBuilder(defId, theType).SetConcurrencyKey("ck-race").Build()
	if err != nil {
		t.Fatalf("build occurrence: %v", err)
	}
	tm := testTenant(t, ctx)
	oe, err := occurrence.ToEntity(om, tm.Id())
	if err != nil {
		t.Fatalf("ToEntity occurrence: %v", err)
	}
	if err := db.WithContext(ctx).Create(&oe).Error; err != nil {
		t.Fatalf("insert occurrence: %v", err)
	}

	m, err := NewBuilder(defId, WorkTypeTriggerEvaluation).SetExecuteAt(time.Now().Add(-time.Minute)).Build()
	if err != nil {
		t.Fatalf("build work: %v", err)
	}
	we, err := ToEntity(m, testTenant(t, ctx))
	if err != nil {
		t.Fatalf("ToEntity work: %v", err)
	}
	if err := db.WithContext(ctx).Create(&we).Error; err != nil {
		t.Fatalf("insert work: %v", err)
	}

	p := NewProcessor(testLogger(t), ctx, db)
	claimed, err := p.ClaimBatch("a", 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %d %v", len(claimed), err)
	}
	if err := p.ExecuteOne(claimed[0]); err != nil {
		t.Fatalf("ExecuteOne: %v", err)
	}

	got := readWork(t, db, we.ID)
	if got.State != StateCompleted {
		t.Fatalf("work state = %s, want COMPLETED (concurrency-key loss must be treated as success)", got.State)
	}
}
