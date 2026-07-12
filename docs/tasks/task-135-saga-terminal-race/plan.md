# Saga Terminal-State Race — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make terminal saga lifecycle states absorbing at event acceptance AND at commit time, route late-successful compensable steps into single-step rollback, and make the Postgres store terminal-preserving — closing the task-102 currency/custody race.

**Architecture:** Layered guards (design §3, approach B): a fast-path lifecycle gate in `AcceptEvent`, an authoritative re-check in `stepCompletedWithResultOnce`, a `version`-bumping `TryTransition` that invalidates every in-flight optimistic write, and terminal-preserving `Put`/`Remove`. Late successes route through a new `Compensator.CompensateLateStep` with a claim-then-dispatch idempotency marker persisted in the saga JSONB blob.

**Tech Stack:** Go, gorm (Postgres prod / sqlite tests), logrus + logtest hooks, OpenTelemetry span (task-040 span-metrics pipeline), testify.

**Module root (all paths below relative to this unless stated):** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`

## Global Constraints

- Verification per CLAUDE.md: `go test -race ./...`, `go test -race -tags=test ./...`, `go vet ./...`, `go build ./...` clean in the module; `docker buildx bake atlas-saga-orchestrator` from the worktree root; `tools/redis-key-guard.sh` from the repo root.
- No new REST/Kafka contract, no new topic, no schema migration (design: marker lives inside the `SagaData` JSONB blob).
- `AcceptEvent` signature unchanged; skip paths return `(AcceptDecision{}, false)` (PRD §4.1/§5).
- Happy path (`pending` lifecycle) behavior unchanged — all existing suites must stay green.
- Late-success rollback is **at-most-once** (claim-then-dispatch): negation inverses (mesos/currency/exp/fame) are not idempotent downstream (design §3.5).
- Lifecycle is never re-transitioned by late compensation; no `Failed` emission from the absorb path (PRD §4.3).
- Test setup uses the project Builder pattern (`NewBuilder()...AddStep(...)`); no `*_testhelpers.go` files.
- Test seams live in `//go:build test` files (existing pattern: `processor_testseam.go`, `producer_testseam.go`).
- No literal home/absolute paths in committed files. No TODO/stub in landed commits.

---

### Task 1: EventKind outcome classification + `SkipReasonSagaTerminal`

**Files:**
- Modify: `saga/event_acceptance.go`
- Test: `saga/event_acceptance_test.go`

**Interfaces:**
- Consumes: existing `EventKind` constants, `acceptanceTable`.
- Produces: `type EventOutcome string`, `OutcomeSuccess`/`OutcomeFailure`, `func EventOutcomeOf(kind EventKind) (EventOutcome, bool)`, `SkipReasonSagaTerminal = "saga_terminal"`. Tasks 5/7/8 depend on these exact names.

- [ ] **Step 1: Write the failing completeness test**

Append to `saga/event_acceptance_test.go` (same guard style as the existing coverage test in that file):

```go
// TestOutcomeTableCompleteness asserts every EventKind referenced anywhere in
// acceptanceTable has an outcome classification. Adding a kind without
// classifying it must fail CI (design §3.1).
func TestOutcomeTableCompleteness(t *testing.T) {
	for action, kinds := range acceptanceTable {
		for _, k := range kinds {
			if _, ok := EventOutcomeOf(k); !ok {
				t.Errorf("EventKind %q (action %q) has no outcomeTable entry", k, action)
			}
		}
	}
}

func TestEventOutcomeOf_FailureKinds(t *testing.T) {
	failures := []EventKind{
		EventKindCharacterCreationFailed,
		EventKindCharacterMesoError,
		EventKindCompartmentCreationFailed,
		EventKindCompartmentError,
		EventKindInventoryCreationFailed,
		EventKindStorageError,
		EventKindStorageCompartmentError,
		EventKindCashShopCompartmentError,
		EventKindInviteRejected,
	}
	for _, k := range failures {
		o, ok := EventOutcomeOf(k)
		assert.True(t, ok, string(k))
		assert.Equal(t, OutcomeFailure, o, string(k))
	}
	o, ok := EventOutcomeOf(EventKindCashShopWalletUpdated)
	assert.True(t, ok)
	assert.Equal(t, OutcomeSuccess, o)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./saga/ -run 'TestOutcomeTable|TestEventOutcomeOf' -v`
Expected: FAIL — `EventOutcomeOf`, `OutcomeFailure`, etc. undefined (compile error).

- [ ] **Step 3: Implement the classification**

In `saga/event_acceptance.go`, after `StepAcceptsEvent`, add:

```go
// EventOutcome classifies an EventKind as a success signal (the step's side
// effect landed downstream) or a failure signal (it did not). Late-after-
// terminal routing (design §3.2/§3.4) uses this to decide whether a rollback
// must be dispatched for an absorbed event.
type EventOutcome string

const (
	OutcomeSuccess EventOutcome = "success"
	OutcomeFailure EventOutcome = "failure"
)

// outcomeTable classifies every declared EventKind. invite.rejected is a
// failure deliberately: a rejected invite left no side effect to roll back.
var outcomeTable = map[EventKind]EventOutcome{
	// Character subsystem.
	EventKindCharacterMapChanged:        OutcomeSuccess,
	EventKindCharacterExperienceChanged: OutcomeSuccess,
	EventKindCharacterLevelChanged:      OutcomeSuccess,
	EventKindCharacterMesoChanged:       OutcomeSuccess,
	EventKindCharacterJobChanged:        OutcomeSuccess,
	EventKindCharacterCreated:           OutcomeSuccess,
	EventKindCharacterCreationFailed:    OutcomeFailure,
	EventKindCharacterStatChanged:       OutcomeSuccess,
	EventKindCharacterMesoError:         OutcomeFailure,
	EventKindCharacterDeleted:           OutcomeSuccess,

	// Asset subsystem.
	EventKindAssetCreated:         OutcomeSuccess,
	EventKindAssetDeleted:         OutcomeSuccess,
	EventKindAssetQuantityChanged: OutcomeSuccess,
	EventKindAssetMoved:           OutcomeSuccess,

	// Quest subsystem.
	EventKindQuestStarted:   OutcomeSuccess,
	EventKindQuestCompleted: OutcomeSuccess,
	EventKindQuestForfeited: OutcomeSuccess,

	// Skill subsystem.
	EventKindSkillCreated: OutcomeSuccess,
	EventKindSkillUpdated: OutcomeSuccess,
	EventKindSkillDeleted: OutcomeSuccess,

	// Buddy list.
	EventKindBuddyCapacityChanged: OutcomeSuccess,

	// Consumable.
	EventKindConsumableEffectApplied: OutcomeSuccess,

	// Pet.
	EventKindPetClosenessChanged: OutcomeSuccess,
	EventKindPetEvolved:          OutcomeSuccess,

	// Cash shop.
	EventKindCashShopWalletUpdated:       OutcomeSuccess,
	EventKindCashShopCompartmentAccepted: OutcomeSuccess,
	EventKindCashShopCompartmentReleased: OutcomeSuccess,
	EventKindCashShopCompartmentError:    OutcomeFailure,

	// Compartment (character inventory).
	EventKindCompartmentCreated:        OutcomeSuccess,
	EventKindCompartmentCreationFailed: OutcomeFailure,
	EventKindCompartmentDeleted:        OutcomeSuccess,
	EventKindCompartmentAccepted:       OutcomeSuccess,
	EventKindCompartmentReleased:       OutcomeSuccess,
	EventKindCompartmentError:          OutcomeFailure,

	// Inventory.
	EventKindInventoryCreated:        OutcomeSuccess,
	EventKindInventoryCreationFailed: OutcomeFailure,

	// Storage.
	EventKindStorageMesosUpdated:        OutcomeSuccess,
	EventKindStorageError:               OutcomeFailure,
	EventKindStorageCompartmentAccepted: OutcomeSuccess,
	EventKindStorageCompartmentReleased: OutcomeSuccess,
	EventKindStorageCompartmentError:    OutcomeFailure,

	// Guild.
	EventKindGuildRequestAgreement: OutcomeSuccess,
	EventKindGuildCreated:          OutcomeSuccess,
	EventKindGuildDisbanded:        OutcomeSuccess,
	EventKindGuildEmblemUpdated:    OutcomeSuccess,
	EventKindGuildCapacityUpdated:  OutcomeSuccess,

	// Invite.
	EventKindInviteCreated:  OutcomeSuccess,
	EventKindInviteAccepted: OutcomeSuccess,
	EventKindInviteRejected: OutcomeFailure,
}

// EventOutcomeOf returns the outcome classification for kind.
func EventOutcomeOf(kind EventKind) (EventOutcome, bool) {
	o, ok := outcomeTable[kind]
	return o, ok
}
```

And add to the `SkipReason*` const block:

```go
	SkipReasonSagaTerminal       = "saga_terminal"
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./saga/ -run 'TestOutcomeTable|TestEventOutcomeOf' -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add saga/event_acceptance.go saga/event_acceptance_test.go
git commit -m "feat(saga-orchestrator): classify event kinds by outcome; add saga_terminal skip reason"
```

---

### Task 2: `lateCompensated` step marker

**Files:**
- Modify: `saga/model.go`
- Test: `saga/model_test.go`

**Interfaces:**
- Produces: `Step[T].LateCompensated() bool`, `Saga.WithStepLateCompensated(index int) (Saga, error)`. JSON key: `lateCompensated` (camelCase, consistent with `stepId`/`createdAt`; design §3.5 wrote the key lowercase — treated as a typo, noted in context.md). Task 5 depends on both names.

- [ ] **Step 1: Write the failing tests**

Append to `saga/model_test.go`:

```go
func TestStepLateCompensatedMarker_RoundTrip(t *testing.T) {
	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
		AddStep("s2", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 1}).
		Build()
	require.NoError(t, err)

	// Default false.
	step, _ := s.StepAt(0)
	assert.False(t, step.LateCompensated())

	// Set on step 0 only.
	s, err = s.WithStepLateCompensated(0)
	require.NoError(t, err)
	s0, _ := s.StepAt(0)
	s1, _ := s.StepAt(1)
	assert.True(t, s0.LateCompensated())
	assert.False(t, s1.LateCompensated())

	// Survives the JSONB persistence path (Marshal → Unmarshal).
	data, err := json.Marshal(s)
	require.NoError(t, err)
	var restored Saga
	require.NoError(t, json.Unmarshal(data, &restored))
	r0, _ := restored.StepAt(0)
	r1, _ := restored.StepAt(1)
	assert.True(t, r0.LateCompensated())
	assert.False(t, r1.LateCompensated())

	// Preserved through WithStepStatus / WithStepResult copies.
	s, err = s.WithStepStatus(0, Completed)
	require.NoError(t, err)
	s0, _ = s.StepAt(0)
	assert.True(t, s0.LateCompensated())
	s, err = s.WithStepResult(0, map[string]any{"k": "v"})
	require.NoError(t, err)
	s0, _ = s.StepAt(0)
	assert.True(t, s0.LateCompensated())

	// Out-of-range index errors.
	_, err = s.WithStepLateCompensated(9)
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./saga/ -run TestStepLateCompensatedMarker_RoundTrip -v`
Expected: FAIL — `LateCompensated`/`WithStepLateCompensated` undefined.

- [ ] **Step 3: Implement the marker**

In `saga/model.go`:

1. Add the field to `Step[T]` (after `result`):

```go
	// lateCompensated records that a single-step rollback was dispatched for
	// this step after the saga went terminal (design §3.5). Claim-then-
	// dispatch idempotency: once set, duplicate late deliveries are no-ops.
	lateCompensated bool
```

2. Add the accessor (after `Result()`):

```go
// LateCompensated reports whether a late-success rollback was already
// dispatched for this step (see Compensator.CompensateLateStep).
func (s Step[T]) LateCompensated() bool { return s.lateCompensated }
```

3. In `Step[T].MarshalJSON`, add to the `alias` struct and literal:

```go
			LateCompensated bool `json:"lateCompensated,omitempty"`
```
```go
			LateCompensated: s.lateCompensated,
```

4. In `Step[T].UnmarshalJSON`, add to the `actionOnly` struct and assignments:

```go
		LateCompensated bool `json:"lateCompensated,omitempty"`
```
```go
	s.lateCompensated = actionOnly.LateCompensated
```

5. Preserve the field in the two copy-on-write step literals — `WithStepStatus` and `WithStepResult` both build `Step[any]{...}` literals; add to **each**:

```go
		lateCompensated: s.steps[index].lateCompensated,
```

Then run `grep -n 'Step\[any\]{' saga/*.go` and add the field to any other literal that copies an existing step's fields (skip literals that construct brand-new steps).

6. Add the copy-on-write setter (after `WithStepResult`):

```go
// WithStepLateCompensated returns a new Saga with the specified step's
// lateCompensated marker set. Mirrors WithStepStatus/WithStepResult.
func (s Saga) WithStepLateCompensated(index int) (Saga, error) {
	if index < 0 || index >= len(s.steps) {
		return Saga{}, fmt.Errorf("invalid step index: %d", index)
	}

	newSteps := make([]Step[any], len(s.steps))
	copy(newSteps, s.steps)

	newSteps[index] = Step[any]{
		stepId:          s.steps[index].stepId,
		status:          s.steps[index].status,
		action:          s.steps[index].action,
		payload:         s.steps[index].payload,
		createdAt:       s.steps[index].createdAt,
		updatedAt:       time.Now(),
		result:          s.steps[index].result,
		lateCompensated: true,
	}

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		timeout:       s.timeout,
		steps:         newSteps,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass (plus the whole model suite)**

Run: `go test ./saga/ -run 'TestStepLateCompensatedMarker_RoundTrip|TestStep|TestSaga' -v`
Expected: PASS, no regressions.

- [ ] **Step 5: Commit**

```bash
git add saga/model.go saga/model_test.go
git commit -m "feat(saga-orchestrator): add lateCompensated step marker with JSON round-trip"
```

---

### Task 3: Terminal-preserving store — version bump on `TryTransition`, guarded `Put`/`Remove`

**Files:**
- Modify: `saga/store.go`
- Test: `saga/store_test.go` (new)

**Interfaces:**
- Consumes: `Entity`, `lifecycleToStatus`, `VersionConflictError` (all existing).
- Produces: behavioral guarantees only — no signature changes. Task 8's retry-absorb and Task 9's integration test depend on: (a) `TryTransition` bumps `version`; (b) `Put` cannot regress `failed`/`completed` status but still updates `saga_data`; (c) `Remove` preserves `failed`.

- [ ] **Step 1: Write the failing store tests**

Create `saga/store_test.go`:

```go
package saga

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Migration(db))
	return db
}

func newStoreTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func newTestStoreSaga(t *testing.T) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
		Build()
	require.NoError(t, err)
	return s
}

// TestPostgresStore_TryTransitionBumpsVersion: an optimistic Put built on a
// pre-terminal read must fail with VersionConflictError once the terminal
// transition commits (design §3.3b).
func TestPostgresStore_TryTransitionBumpsVersion(t *testing.T) {
	logger, _ := test.NewNullLogger()
	store := NewPostgresStore(newStoreTestDB(t), logger)
	ctx := newStoreTestCtx(t)
	s := newTestStoreSaga(t)

	require.NoError(t, store.Put(ctx, s))
	_, ok := store.GetById(ctx, s.TransactionId()) // tracks version 1
	require.True(t, ok)

	require.True(t, store.TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))

	err := store.Put(ctx, s) // built on the stale (pre-transition) version
	var vce *VersionConflictError
	require.ErrorAs(t, err, &vce)
}

// TestPostgresStore_PutCannotResurrectTerminal: a Put built on a FRESH read of
// a terminal saga updates saga_data but cannot regress status (design §3.3c).
func TestPostgresStore_PutCannotResurrectTerminal(t *testing.T) {
	logger, _ := test.NewNullLogger()
	store := NewPostgresStore(newStoreTestDB(t), logger)
	ctx := newStoreTestCtx(t)
	s := newTestStoreSaga(t)

	require.NoError(t, store.Put(ctx, s))
	require.True(t, store.TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))
	require.True(t, store.TryTransition(ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed))

	fresh, ok := store.GetById(ctx, s.TransactionId()) // re-tracks post-bump version
	require.True(t, ok)
	marked, err := fresh.WithStepLateCompensated(0)
	require.NoError(t, err)
	require.NoError(t, store.Put(ctx, marked)) // succeeds: fresh version

	lc, ok := store.GetLifecycle(ctx, s.TransactionId())
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleFailed, lc, "Put must not regress failed status")

	reread, ok := store.GetById(ctx, s.TransactionId())
	require.True(t, ok)
	st, _ := reread.StepAt(0)
	assert.True(t, st.LateCompensated(), "saga_data update must still land")
}

// TestPostgresStore_RemovePreservesFailed: Remove collapses active/compensating
// to completed but must not erase the failed audit state (design defect 3).
func TestPostgresStore_RemovePreservesFailed(t *testing.T) {
	logger, _ := test.NewNullLogger()
	store := NewPostgresStore(newStoreTestDB(t), logger)
	ctx := newStoreTestCtx(t)

	failed := newTestStoreSaga(t)
	require.NoError(t, store.Put(ctx, failed))
	require.True(t, store.TryTransition(ctx, failed.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))
	require.True(t, store.TryTransition(ctx, failed.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed))
	assert.True(t, store.Remove(ctx, failed.TransactionId()))
	lc, ok := store.GetLifecycle(ctx, failed.TransactionId())
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleFailed, lc)

	active := newTestStoreSaga(t)
	require.NoError(t, store.Put(ctx, active))
	assert.True(t, store.Remove(ctx, active.TransactionId()))
	lc, ok = store.GetLifecycle(ctx, active.TransactionId())
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleCompleted, lc)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./saga/ -run TestPostgresStore_ -v`
Expected: `TestPostgresStore_TryTransitionBumpsVersion` FAILS (Put succeeds — no version bump today); `TestPostgresStore_RemovePreservesFailed` FAILS (failed collapses to completed). `TestPostgresStore_PutCannotResurrectTerminal` FAILS (lifecycle regresses to `pending`/`active`).

If `gorm.io/driver/sqlite` is not yet importable, run `go mod tidy` first (it is already an indirect dependency).

- [ ] **Step 3: Implement the store guards**

In `saga/store.go`:

1. **`TryTransition`** — bump the version in the same atomic UPDATE. Replace the `Updates` map with:

```go
		Updates(map[string]interface{}{
			"status":     toStatus,
			"version":    gorm.Expr("version + 1"),
			"updated_at": time.Now(),
		})
```

Add above the update (comment, not code):

```go
	// The version bump deliberately does NOT touch s.ver: every optimistic
	// Put built on a pre-transition read — on this instance or any other —
	// must fail with VersionConflictError and re-read, at which point the
	// stepCompletedWithResultOnce terminal gate absorbs (design §3.3b).
	// Syncing or deleting the local entry here would let such a Put slip
	// through the unguarded insert/OnConflict path.
```

2. **`Put` optimistic branch** — make the status assignment terminal-preserving. Replace `"status": sagaStatus,` with:

```go
				// Terminal-preserving: failed/completed can never be
				// overwritten by a recomputed active/compensating status;
				// saga_data still updates (late-compensation marker).
				"status": gorm.Expr("CASE WHEN status IN ('failed','completed') THEN status ELSE ? END", sagaStatus),
```

3. **`Put` insert branch** — the `OnConflict` clause today overwrites status/version unconditionally. Replace `DoUpdates: clause.AssignmentColumns(...)` with:

```go
			DoUpdates: clause.Assignments(map[string]interface{}{
				"saga_type":    string(saga.SagaType()),
				"initiated_by": saga.InitiatedBy(),
				"status":       gorm.Expr("CASE WHEN sagas.status IN ('failed','completed') THEN sagas.status ELSE excluded.status END"),
				"saga_data":    data,
				"version":      gorm.Expr("sagas.version + 1"),
				"updated_at":   time.Now(),
			}),
```

(`excluded` and table-qualified column references are valid in both Postgres and sqlite upserts. The monotonic `sagas.version + 1` replaces the old reset-to-1; the locally tracked `ver=1` after this path may mismatch a conflict-updated row, which fails closed — the next Put conflicts and re-reads.)

4. **`Remove`** — preserve `failed`. Replace `"status": "completed",` with:

```go
			// failed is a terminal audit state; do not mask a timed-out saga
			// as completed (design defect 3).
			"status": gorm.Expr("CASE WHEN status = 'failed' THEN status ELSE 'completed' END"),
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./saga/ -run TestPostgresStore_ -v`
Expected: PASS (all three).

- [ ] **Step 5: Run the full package to catch regressions**

Run: `go test -race ./saga/... && go test -race -tags=test ./saga/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add saga/store.go saga/store_test.go
git commit -m "fix(saga-orchestrator): version-bump TryTransition; terminal-preserving Put/Remove"
```

---

### Task 4: Cashshop processor injection plumbing + cashshop mock

**Files:**
- Create: `cashshop/mock/processor.go`
- Modify: `saga/compensator.go` (csP field + `WithCashshopProcessor`)
- Modify: `saga/processor.go` (`Processor.WithCashshopProcessor`)
- Modify: `saga/mock/processor.go` (new interface method)

**Interfaces:**
- Consumes: `cashshop.Processor` (6 methods: `AwardCurrencyAndEmit`, `AwardCurrency`, `AcceptAndEmit`, `Accept`, `ReleaseAndEmit`, `Release` — see `cashshop/processor.go:14`).
- Produces: `Compensator.WithCashshopProcessor(cashshop.Processor) Compensator`, `Processor.WithCashshopProcessor(cashshop.Processor) Processor`, `cashshop/mock.ProcessorMock`. Tasks 5/7/9 depend on these. Naming matches the existing `Handler.WithCashshopProcessor` spelling (`handler.go:63`).

No new behavior to TDD here — this is dependency plumbing verified by compilation plus the existing suites; Task 5's tests exercise it.

- [ ] **Step 1: Create the cashshop mock**

Create `cashshop/mock/processor.go`, mirroring `compartment/mock/processor.go`:

```go
package mock

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the cashshop.Processor interface.
type ProcessorMock struct {
	AwardCurrencyAndEmitFunc func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AwardCurrencyFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AcceptAndEmitFunc        func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error
	AcceptFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error
	ReleaseAndEmitFunc       func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error
	ReleaseFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error
}

func (m *ProcessorMock) AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	if m.AwardCurrencyAndEmitFunc != nil {
		return m.AwardCurrencyAndEmitFunc(transactionId, accountId, currencyType, amount)
	}
	return nil
}

func (m *ProcessorMock) AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	if m.AwardCurrencyFunc != nil {
		return m.AwardCurrencyFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, int32) error { return nil }
}

func (m *ProcessorMock) AcceptAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
	if m.AcceptAndEmitFunc != nil {
		return m.AcceptAndEmitFunc(transactionId, characterId, accountId, compartmentId, compartmentType, cashId, templateId, quantity, commodityId, purchasedBy, flag)
	}
	return nil
}

func (m *ProcessorMock) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, uuid.UUID, byte, int64, uint32, uint32, uint32, uint32, uint16) error { return nil }
}

func (m *ProcessorMock) ReleaseAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
	if m.ReleaseAndEmitFunc != nil {
		return m.ReleaseAndEmitFunc(transactionId, characterId, accountId, compartmentId, compartmentType, assetId, cashId, templateId)
	}
	return nil
}

func (m *ProcessorMock) Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, uuid.UUID, byte, uint32, int64, uint32) error { return nil }
}
```

(Adjust exact signatures against `cashshop/processor.go` if they drift — the interface is the source of truth. Verify with a `var _ cashshop.Processor = (*ProcessorMock)(nil)` assertion in `cashshop/mock/processor_test.go`, mirroring the sibling mocks.)

- [ ] **Step 2: Thread cashshop through the Compensator**

In `saga/compensator.go`:

1. Add `"atlas-saga-orchestrator/cashshop"` to imports.
2. Add to the `Compensator` interface, after `WithInviteProcessor`:

```go
	WithCashshopProcessor(cashshop.Processor) Compensator
```

3. Add field `csP cashshop.Processor` to `CompensatorImpl` and initialize it in `NewCompensator` with `cashshop.NewProcessor(l, ctx)` (alongside the existing processor constructions).
4. Add the builder method, mirroring the existing `With*Processor` methods on `CompensatorImpl` (copy all fields, replace `csP`):

```go
func (c *CompensatorImpl) WithCashshopProcessor(csP cashshop.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   c.charP,
		compP:   c.compP,
		skillP:  c.skillP,
		validP:  c.validP,
		guildP:  c.guildP,
		inviteP: c.inviteP,
		csP:     csP,
	}
}
```

5. **Every existing `With*Processor` method on `CompensatorImpl` must now also copy `csP`** (they build full struct literals). Add `csP: c.csP,` to each.

- [ ] **Step 3: Thread cashshop through the Processor**

In `saga/processor.go`:

1. Add to the `Processor` interface, after `WithInviteProcessor`:

```go
	WithCashshopProcessor(cashshop.Processor) Processor
```

2. Add the method on `ProcessorImpl`, mirroring `WithInviteProcessor` (`processor.go:198`) — thread into both `comp` and `handle` (the handler already supports it):

```go
func (p *ProcessorImpl) WithCashshopProcessor(csP cashshop.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithCashshopProcessor(csP),
		handle:  p.handle.WithCashshopProcessor(csP),
		charP:   p.charP,
		compP:   p.compP,
		skillP:  p.skillP,
		validP:  p.validP,
		guildP:  p.guildP,
		inviteP: p.inviteP,
	}
}
```

3. `"atlas-saga-orchestrator/cashshop"` import.

- [ ] **Step 4: Update the saga processor mock**

In `saga/mock/processor.go`, add (mirroring the existing `With*ProcessorFunc` fields and methods):

```go
	WithCashshopProcessorFunc func(cashshop.Processor) saga.Processor
```
```go
func (m *ProcessorMock) WithCashshopProcessor(p cashshop.Processor) saga.Processor {
	if m.WithCashshopProcessorFunc != nil {
		return m.WithCashshopProcessorFunc(p)
	}
	return m
}
```

plus the `"atlas-saga-orchestrator/cashshop"` import.

- [ ] **Step 5: Verify compilation and existing suites**

Run: `go build ./... && go test -race ./... && go test -race -tags=test ./...`
Expected: clean build, all existing tests PASS.

- [ ] **Step 6: Commit**

```bash
git add cashshop/mock/ saga/compensator.go saga/processor.go saga/mock/processor.go
git commit -m "feat(saga-orchestrator): thread cashshop processor through compensator/processor; add cashshop mock"
```

---

### Task 5: `Compensator.CompensateLateStep` — claim-then-dispatch single-step rollback

**Files:**
- Modify: `saga/compensator.go`
- Test: `saga/compensator_test.go`

**Interfaces:**
- Consumes: Task 1 (`OutcomeSuccess`), Task 2 (`LateCompensated`, `WithStepLateCompensated`), Task 4 (`csP`), existing `isVersionConflict`, `maxConflictRetries`, `ExtractCharacterCreationIds`, `extractCharacterCreationWorldId`, payload types from `libs/atlas-saga` and `saga/model.go`.
- Produces: `CompensateLateStep(s Saga, step Step[any]) (bool, error)` on the `Compensator` interface — returns `compensated=true` only when an inverse command was dispatched this call. (Design §3.4 showed `error` only; the boolean is required by §3.6's `late.compensated` span attribute — deviation noted in context.md.) Tasks 7/8/9 depend on this exact signature.

- [ ] **Step 1: Write the failing tests**

Append to `saga/compensator_test.go` (imports: `cashshopmock "atlas-saga-orchestrator/cashshop/mock"`, `compartmentmock "atlas-saga-orchestrator/compartment/mock"`, logtest, tenant, uuid, testify — follow the file's existing conventions):

```go
func lateStepTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

// The task-102 shape: a late-successful AwardCurrency step dispatches exactly
// one negated wallet credit, and a duplicate delivery dispatches nothing.
func TestCompensateLateStep_AwardCurrency_NegatedOnceOnly(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 100}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	var calls []int32
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(txId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
			assert.Equal(t, s.TransactionId(), txId)
			assert.Equal(t, uint32(42), accountId)
			assert.Equal(t, uint32(2), currencyType)
			calls = append(calls, amount)
			return nil
		},
	}
	c := NewCompensator(logger, ctx).WithCashshopProcessor(cs)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	require.Len(t, calls, 1)
	assert.Equal(t, int32(-100), calls[0], "inverse must negate the amount")

	// Duplicate delivery: marker already claimed — no second dispatch.
	fresh, ok := GetCache().GetById(ctx, s.TransactionId())
	require.True(t, ok)
	freshStep, _ := fresh.GetCurrentStep()
	assert.True(t, freshStep.LateCompensated())
	compensated, err = c.CompensateLateStep(fresh, freshStep)
	require.NoError(t, err)
	assert.False(t, compensated)
	assert.Len(t, calls, 1)
}

func TestCompensateLateStep_AwardAsset_DestroysItem(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_item", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 7, Item: ItemPayload{TemplateId: 2000000, Quantity: 3}}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	destroyed := 0
	cp := &compartmentmock.ProcessorMock{
		RequestDestroyItemFunc: func(txId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
			destroyed++
			assert.Equal(t, uint32(7), characterId)
			assert.Equal(t, uint32(2000000), templateId)
			assert.Equal(t, uint32(3), quantity)
			return nil
		},
	}
	c := NewCompensator(logger, ctx).WithCompartmentProcessor(cp)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	assert.Equal(t, 1, destroyed)
}

// Non-compensable action: absorb-only with a late_effect_unrecoverable WARN.
func TestCompensateLateStep_NonCompensable_WarnsNoDispatch(t *testing.T) {
	ResetCache()
	logger, hook := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("change_hair", Pending, ChangeHair, ChangeHairPayload{CharacterId: 1}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	c := NewCompensator(logger, ctx)
	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.False(t, compensated)

	var warned bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["reason"] == "late_effect_unrecoverable" {
			warned = true
		}
	}
	assert.True(t, warned, "expected late_effect_unrecoverable WARN")

	// Marker must NOT be claimed for a non-compensable step.
	fresh, _ := GetCache().GetById(ctx, s.TransactionId())
	freshStep, _ := fresh.GetCurrentStep()
	assert.False(t, freshStep.LateCompensated())
}
```

(If `ChangeHairPayload`'s field set differs, construct it with whatever minimal fields `saga/model.go` / `libs/atlas-saga/payloads.go` define — the action choice is what matters.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -race ./saga/ -run TestCompensateLateStep -v`
Expected: FAIL — `CompensateLateStep` undefined.

- [ ] **Step 3: Implement `CompensateLateStep`**

In `saga/compensator.go`:

1. Add to the `Compensator` interface:

```go
	// CompensateLateStep dispatches the single-step inverse for a step whose
	// success event arrived after the saga went terminal (PRD §4.3, design
	// §3.4/§3.5). Pure dispatch — no lifecycle transitions, no Failed
	// emission, no cache eviction. Claim-then-dispatch: the lateCompensated
	// marker is persisted BEFORE the inverse goes out, giving at-most-once
	// rollback, because the negation inverses (mesos/currency/exp/fame) are
	// not idempotent downstream — at-least-once would double-refund. A crash
	// between claim and dispatch loses the rollback but is auditable via the
	// saga_terminal log + span emitted by the caller. Returns true only when
	// an inverse command was dispatched by this call.
	CompensateLateStep(s Saga, step Step[any]) (bool, error)
```

2. Implementation (new imports: `character2 "atlas-saga-orchestrator/kafka/message/character"`; `errors` if not present):

```go
// lateCompensableActions is the v1 compensable set (design §3.4): the full
// value-transfer class that broke the task-102 invariant. Everything else is
// absorb-only and logged as late_effect_unrecoverable when hit.
// DestroyAssetFromSlot is deliberately absent: its payload carries no
// TemplateId, so the destroyed item cannot be recreated from the step alone.
var lateCompensableActions = map[Action]struct{}{
	AwardAsset:            {},
	CreateAndEquipAsset:   {},
	CreateSkill:           {},
	CreateCharacter:       {},
	AwaitCharacterCreated: {},
	DestroyAsset:          {},
	AwardMesos:            {},
	AwardCurrency:         {},
	AwardExperience:       {},
	DeductExperience:      {},
	AwardFame:             {},
	EquipAsset:            {},
	UnequipAsset:          {},
}

func (c *CompensatorImpl) CompensateLateStep(s Saga, step Step[any]) (bool, error) {
	fields := logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        step.StepId(),
		"step_action":    step.Action(),
		"tenant_id":      c.t.Id().String(),
	}

	if _, ok := lateCompensableActions[step.Action()]; !ok {
		fields["reason"] = "late_effect_unrecoverable"
		c.l.WithFields(fields).Warn("Late-successful step has no registered inverse; its effect is orphaned.")
		return false, nil
	}

	claimed, err := c.claimLateCompensation(s.TransactionId(), step.StepId())
	if err != nil {
		return false, err
	}
	if !claimed {
		c.l.WithFields(fields).Debug("Late-success compensation already claimed; duplicate delivery ignored.")
		return false, nil
	}

	if err := c.dispatchLateInverse(s, step); err != nil {
		// The claim is already persisted: at-most-once means we do NOT retry
		// dispatch on a later redelivery. Log loudly for the audit trail.
		fields["reason"] = "late_effect_dispatch_failed"
		c.l.WithFields(fields).WithError(err).Error("Late-success inverse dispatch failed after claim.")
		return true, err
	}

	fields["reason"] = "late_effect_compensated"
	c.l.WithFields(fields).Info("Late-successful step routed into compensation; effect rolled back.")
	return true, nil
}

// claimLateCompensation atomically sets the step's lateCompensated marker.
// Returns false when the marker was already set (duplicate delivery). Only
// the goroutine whose Put wins the optimistic-version race proceeds to
// dispatch; losers re-read and observe the marker.
func (c *CompensatorImpl) claimLateCompensation(transactionId uuid.UUID, stepId string) (bool, error) {
	for attempt := 1; attempt <= maxConflictRetries; attempt++ {
		s, ok := GetCache().GetById(c.ctx, transactionId)
		if !ok {
			return false, errors.New("saga not found while claiming late compensation")
		}
		index := -1
		for i, st := range s.Steps() {
			if st.StepId() == stepId {
				index = i
				break
			}
		}
		if index == -1 {
			return false, fmt.Errorf("step [%s] not found while claiming late compensation", stepId)
		}
		st, _ := s.StepAt(index)
		if st.LateCompensated() {
			return false, nil
		}
		updated, err := s.WithStepLateCompensated(index)
		if err != nil {
			return false, err
		}
		err = GetCache().Put(c.ctx, updated)
		if err == nil {
			return true, nil
		}
		if !isVersionConflict(err) {
			return false, err
		}
	}
	return false, fmt.Errorf("max retries exceeded claiming late compensation for saga %s", transactionId.String())
}

// dispatchLateInverse fires the single-step inverse computed from the STEP
// payload (never the event payload), reusing the reverse-walk idioms.
func (c *CompensatorImpl) dispatchLateInverse(s Saga, step Step[any]) error {
	switch step.Action() {
	case AwardAsset:
		payload, ok := step.Payload().(AwardItemActionPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardAsset compensation")
		}
		return c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false)
	case CreateAndEquipAsset:
		payload, ok := step.Payload().(CreateAndEquipAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late CreateAndEquipAsset compensation")
		}
		return c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false)
	case CreateSkill:
		payload, ok := step.Payload().(CreateSkillPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late CreateSkill compensation")
		}
		return c.skillP.RequestDeleteSkill(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.SkillId)
	case CreateCharacter, AwaitCharacterCreated:
		_, characterId := ExtractCharacterCreationIds(s)
		worldId := extractCharacterCreationWorldId(s)
		if characterId == 0 {
			return fmt.Errorf("late character-creation compensation: character id unresolved")
		}
		return c.charP.RequestDeleteCharacter(s.TransactionId(), characterId, worldId)
	case DestroyAsset:
		payload, ok := step.Payload().(DestroyAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late DestroyAsset compensation")
		}
		qty := payload.Quantity
		if qty == 0 {
			qty = 1
		}
		return c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{})
	case AwardMesos:
		payload, ok := step.Payload().(AwardMesosPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardMesos compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false)
	case AwardCurrency:
		payload, ok := step.Payload().(AwardCurrencyPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardCurrency compensation")
		}
		return c.csP.AwardCurrencyAndEmit(s.TransactionId(), payload.AccountId, payload.CurrencyType, -payload.Amount)
	case AwardExperience:
		payload, ok := step.Payload().(AwardExperiencePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardExperience compensation")
		}
		var total uint32
		for _, d := range payload.Distributions {
			total += d.Amount
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.DeductExperienceAndEmit(s.TransactionId(), ch, payload.CharacterId, total)
	case DeductExperience:
		payload, ok := step.Payload().(DeductExperiencePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late DeductExperience compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardExperienceAndEmit(s.TransactionId(), ch, payload.CharacterId,
			[]character2.ExperienceDistributions{{ExperienceType: "WHITE", Amount: payload.Amount}}, false)
	case AwardFame:
		payload, ok := step.Payload().(AwardFamePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardFame compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardFameAndEmit(s.TransactionId(), ch, payload.CharacterId, -payload.Amount)
	case EquipAsset:
		payload, ok := step.Payload().(EquipAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late EquipAsset compensation")
		}
		return c.compP.RequestUnequipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	case UnequipAsset:
		payload, ok := step.Payload().(UnequipAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late UnequipAsset compensation")
		}
		return c.compP.RequestEquipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	}
	return fmt.Errorf("no late inverse registered for action %s", step.Action())
}
```

(Signatures verified against source: `RequestDestroyItem`/`RequestCreateItem` `compartment/processor.go:32-34`, `RequestEquipAsset`/`RequestUnequipAsset` per `compensator.go:316,379`, `AwardMesosAndEmit`/`DeductExperienceAndEmit`/`AwardExperienceAndEmit`/`AwardFameAndEmit` `character/processor.go:26-35`, `RequestDeleteSkill` `skill/processor.go:22`, `RequestDeleteCharacter` `character/processor.go:45`, `AwardCurrencyAndEmit` `cashshop/processor.go:15`. `ExperienceType: "WHITE"` matches the standard award construction in `services/atlas-messages/.../commands.go:99`.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./saga/ -run TestCompensateLateStep -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add saga/compensator.go saga/compensator_test.go
git commit -m "feat(saga-orchestrator): claim-then-dispatch late-step compensation (CompensateLateStep)"
```

---

### Task 6: `EmitSagaFailed` test seam

**Files:**
- Modify: `saga/producer.go`
- Modify: `saga/producer_testseam.go`

**Interfaces:**
- Produces: `SetEmitSagaFailedForTest(fn) (restore fn)` — Task 9 uses it to count Failed emissions. Follows the existing `emitConversationRewardNoticeFn` pattern exactly.

- [ ] **Step 1: Refactor `EmitSagaFailedByIds` behind a swappable var**

In `saga/producer.go`, replace the body wiring (public functions and doc comments unchanged):

```go
// emitSagaFailedByIdsFn is swappable in tests (SetEmitSagaFailedForTest) so
// integration tests can count Failed emissions without Kafka.
var emitSagaFailedByIdsFn = emitSagaFailedByIdsImpl

func EmitSagaFailedByIds(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, accountId, characterId uint32, errorCode, reason, failedStep string) error {
	return emitSagaFailedByIdsFn(l, ctx, transactionId, sagaType, accountId, characterId, errorCode, reason, failedStep)
}

func emitSagaFailedByIdsImpl(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, sagaType string, accountId, characterId uint32, errorCode, reason, failedStep string) error {
	return producer.ProviderImpl(l)(ctx)(saga.EnvStatusEventTopic)(
		FailedStatusEventProvider(transactionId, accountId, characterId, sagaType, errorCode, reason, failedStep),
	)
}
```

- [ ] **Step 2: Add the test seam**

Append to `saga/producer_testseam.go` (already `//go:build test`):

```go
// SetEmitSagaFailedForTest swaps the underlying Failed-emission function and
// returns the previous one for restoration. Compiled only with -tags=test.
func SetEmitSagaFailedForTest(fn func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error) func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error {
	prev := emitSagaFailedByIdsFn
	emitSagaFailedByIdsFn = fn
	return prev
}
```

(Add the `uuid` import to the seam file.)

- [ ] **Step 3: Verify compilation and suites**

Run: `go build ./... && go test -race ./saga/... && go test -race -tags=test ./saga/...`
Expected: clean, all PASS.

- [ ] **Step 4: Commit**

```bash
git add saga/producer.go saga/producer_testseam.go
git commit -m "test(saga-orchestrator): add EmitSagaFailed test seam"
```

---

### Task 7: Terminal gate in `AcceptEvent` + absorb core + observability span

**Files:**
- Modify: `saga/processor.go`
- Modify: `go.mod` (otel promoted to direct via `go mod tidy`)
- Test: `saga/accept_event_test.go`

**Interfaces:**
- Consumes: Task 1 (`EventOutcomeOf`, `SkipReasonSagaTerminal`), Task 5 (`CompensateLateStep`).
- Produces: `absorbLateTerminal(s Saga, lc SagaLifecycleState, eventKind string, outcome EventOutcome, step Step[any], matched bool)` — Task 8 reuses it. Span name `saga.late_event_absorbed`, attributes `tenant.id`, `saga.type`, `saga.lifecycle_state`, `late.outcome`, `late.compensated` (design §3.6).

- [ ] **Step 1: Write the failing gate tests**

Append to `saga/accept_event_test.go` (reuse the file's `newAcceptEventTestProcessor` / `putAcceptEventSaga` helpers):

```go
// terminalLifecycle drives the saga to the requested terminal state via the
// legal transition chain.
func terminalLifecycle(t *testing.T, ctx context.Context, tx uuid.UUID, target SagaLifecycleState) {
	t.Helper()
	switch target {
	case SagaLifecycleCompensating:
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))
	case SagaLifecycleFailed:
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecycleCompensating, SagaLifecycleFailed))
	case SagaLifecycleCompleted:
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompleted))
	default:
		t.Fatalf("not a terminal state: %s", target)
	}
}

func TestAcceptEvent_TerminalLifecycleAbsorbs(t *testing.T) {
	for _, terminal := range []SagaLifecycleState{SagaLifecycleCompensating, SagaLifecycleFailed, SagaLifecycleCompleted} {
		t.Run(string(terminal), func(t *testing.T) {
			ResetCache()
			p, hook, ctx := newAcceptEventTestProcessor(t)
			tx := uuid.New()
			s, err := NewBuilder().
				SetTransactionId(tx).
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
				Build()
			require.NoError(t, err)
			putAcceptEventSaga(t, ctx, s)
			terminalLifecycle(t, ctx, tx, terminal)

			// Pending, action-matching step present — pre-fix this advanced the saga.
			_, ok := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
			assert.False(t, ok, "terminal lifecycle must absorb the event")

			var entry *logrus.Entry
			for _, e := range hook.AllEntries() {
				if e.Data["reason"] == SkipReasonSagaTerminal {
					entry = e
				}
			}
			require.NotNil(t, entry, "expected saga_terminal skip log")
			assert.Equal(t, tx.String(), entry.Data["transaction_id"])
			assert.Equal(t, string(EventKindCashShopWalletUpdated), entry.Data["event_kind"])
			assert.Equal(t, string(terminal), entry.Data["lifecycle_state"])
			assert.Equal(t, "award_currency_seller", entry.Data["step_id"])
		})
	}
}

func TestAcceptEvent_PendingLifecycleStillAccepts(t *testing.T) {
	ResetCache()
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	assert.True(t, ok, "pending lifecycle is unchanged happy path")
	assert.Equal(t, "award_currency_seller", decision.Step.StepId())
}

// A late SUCCESS event for a compensable step routes into CompensateLateStep;
// a late FAILURE event absorbs without compensation (PRD §4.3).
func TestAcceptEvent_TerminalRoutesLateSuccessOnly(t *testing.T) {
	ResetCache()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := acceptEventTestCtx(t)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_mesos", Pending, AwardMesos, AwardMesosPayload{CharacterId: 1, Amount: 500}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))
	terminalLifecycle(t, ctx, tx, SagaLifecycleFailed)

	refunds := 0
	charMock := &charactermock.ProcessorMock{
		AwardMesosAndEmitFunc: func(txId uuid.UUID, ch channel.Model, characterId uint32, actorId uint32, actorType string, amount int32, showEffect bool) error {
			refunds++
			assert.Equal(t, int32(-500), amount)
			return nil
		},
	}
	p := NewProcessor(logger, ctx).WithCharacterProcessor(charMock)

	// Failure-outcome kind for the same action: absorb-only.
	_, ok := p.AcceptEvent(tx, EventKindCharacterMesoError)
	assert.False(t, ok)
	assert.Equal(t, 0, refunds, "failure outcome must not compensate")

	// Success-outcome kind: absorb + route into compensation.
	_, ok = p.AcceptEvent(tx, EventKindCharacterMesoChanged)
	assert.False(t, ok)
	assert.Equal(t, 1, refunds, "success outcome must dispatch exactly one inverse")
}
```

(Imports to add in the test file: `charactermock "atlas-saga-orchestrator/character/mock"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/channel"`. If the character mock lacks `AwardMesosAndEmitFunc`, add it there following the file's pattern.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -race ./saga/ -run 'TestAcceptEvent_Terminal|TestAcceptEvent_Pending' -v`
Expected: `TestAcceptEvent_TerminalLifecycleAbsorbs` and `TestAcceptEvent_TerminalRoutesLateSuccessOnly` FAIL (event accepted / no skip log); `TestAcceptEvent_PendingLifecycleStillAccepts` PASSES (guards the happy path).

- [ ] **Step 3: Implement the gate, absorb core, and span**

In `saga/processor.go`:

1. Imports: add `"go.opentelemetry.io/otel"` and `"go.opentelemetry.io/otel/attribute"`.

2. In `AcceptEvent`, insert between the saga-not-found check and the `GetCurrentStep` check (PRD §4.1 ordering — before no-pending-step / action-mismatch):

```go
	// Terminal lifecycle states are absorbing (PRD §4.1): a saga the timer
	// or a failure path has already moved to compensating/failed/completed
	// can never be advanced by a late step event. A cache miss on
	// GetLifecycle (hard-deleted in-memory entry racing GetById) falls
	// through to the existing checks.
	if lc, ok := GetCache().GetLifecycle(p.ctx, transactionId); ok && lc != SagaLifecyclePending {
		p.absorbLateTerminalEvent(s, lc, kind)
		return AcceptDecision{}, false
	}
```

3. Add the absorb functions (near `maybeWarnUnmatchedEvent`):

```go
// absorbLateTerminalEvent handles a step event that arrived after the saga's
// lifecycle went terminal. The event never advances the saga; a success-
// outcome event for a compensable in-flight step is routed into single-step
// compensation so its real side effect is rolled back (PRD §4.2/§4.3).
func (p *ProcessorImpl) absorbLateTerminalEvent(s Saga, lc SagaLifecycleState, kind EventKind) {
	// Unknown kinds default to failure: never dispatch a rollback for an
	// effect we cannot classify.
	outcome := OutcomeFailure
	if o, ok := EventOutcomeOf(kind); ok {
		outcome = o
	}
	// Steps dispatch serially, so the earliest pending step IS the in-flight
	// one; match it exactly as the happy path would (design §3.2).
	step, stepOk := s.GetCurrentStep()
	matched := stepOk && StepAcceptsEvent(step.Action(), kind)
	p.absorbLateTerminal(s, lc, string(kind), outcome, step, matched)
}

// absorbLateTerminal is the shared core for the AcceptEvent fast-path gate
// and the stepCompletedWithResultOnce commit-time gate.
func (p *ProcessorImpl) absorbLateTerminal(s Saga, lc SagaLifecycleState, eventKind string, outcome EventOutcome, step Step[any], matched bool) {
	fields := logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"event_kind":      eventKind,
		"lifecycle_state": string(lc),
		"saga_type":       s.SagaType(),
		"tenant_id":       p.t.Id().String(),
	}
	if matched {
		fields["step_id"] = step.StepId()
	}
	LogSkip(p.l, fields, SkipReasonSagaTerminal)

	compensated := false
	if matched && outcome == OutcomeSuccess {
		var err error
		compensated, err = p.comp.CompensateLateStep(s, step)
		if err != nil {
			p.l.WithFields(fields).WithError(err).Error("Late-success compensation failed.")
		}
	}

	// task-040 span-metrics pipeline: the counter is
	// traces_spanmetrics_calls_total{span_name="saga.late_event_absorbed"}.
	// transaction.id is on the forbidden-attribute list; it lives in the log
	// line above instead.
	_, span := otel.GetTracerProvider().Tracer("atlas-saga-orchestrator").Start(p.ctx, "saga.late_event_absorbed")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.String("saga.type", string(s.SagaType())),
		attribute.String("saga.lifecycle_state", string(lc)),
		attribute.String("late.outcome", string(outcome)),
		attribute.Bool("late.compensated", compensated),
	)
	span.End()
}
```

4. Run `go mod tidy` in the module so otel moves from `// indirect` to direct.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./saga/ -run TestAcceptEvent -v`
Expected: PASS — all new tests plus all pre-existing `TestAcceptEvent_*` (saga-not-found, no-pending-step, action-mismatch behavior unchanged).

- [ ] **Step 5: Commit**

```bash
git add saga/processor.go saga/accept_event_test.go go.mod go.sum
git commit -m "feat(saga-orchestrator): absorb late events at AcceptEvent when saga lifecycle is terminal"
```

---

### Task 8: Commit-time gate in `stepCompletedWithResultOnce` + ordering-invariant doc

**Files:**
- Modify: `saga/processor.go:424` (`stepCompletedWithResultOnce`)
- Modify: `saga/lifecycle.go` (invariant doc comment)
- Test: `saga/processor_test.go`

**Interfaces:**
- Consumes: Task 7 (`absorbLateTerminal`), Task 1 (`OutcomeSuccess`/`OutcomeFailure`).
- Produces: the TOCTOU closure — with Task 3's version bump, every forward write built on a pre-terminal read retries into this gate.

- [ ] **Step 1: Write the failing TOCTOU test**

Append to `saga/processor_test.go`:

```go
// TestStepCompleted_TerminalAfterAccept reproduces the TOCTOU interleave
// (design §3.3a / walkthrough §4.4): the event passes AcceptEvent while the
// saga is pending, the timeout commits terminal, and only then does the
// handler call StepCompleted(true). The completion must absorb-and-route,
// never advance the step walk.
func TestStepCompleted_TerminalAfterAccept(t *testing.T) {
	ResetCache()
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 100}).
		AddStep("move_item", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 1, Item: ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	refunds := 0
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(txId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
			refunds++
			assert.Equal(t, int32(-100), amount)
			return nil
		},
	}
	forward := 0
	cp := &compartmentmock.ProcessorMock{
		RequestCreateItemWithStatsFunc: func(uuid.UUID, uint32, uint32, uint32, time.Time, bool) error {
			forward++ // step-2 forward dispatch — must never fire
			return nil
		},
	}
	p := NewProcessor(logger, ctx).WithCashshopProcessor(cs).WithCompartmentProcessor(cp)

	// 1. Event accepted while pending.
	_, ok := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	require.True(t, ok)

	// 2. Timeout commits terminal between accept and completion.
	require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))
	require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecycleCompensating, SagaLifecycleFailed))

	// 3. Handler-side completion arrives late.
	require.NoError(t, p.StepCompleted(tx, true))

	// No forward advance: step 1 still pending, step 2 never dispatched.
	fresh, found := GetCache().GetById(ctx, tx)
	require.True(t, found)
	st, _ := fresh.StepAt(0)
	assert.Equal(t, Pending, st.Status(), "forward write must not land")
	assert.Equal(t, 0, forward, "next step must not dispatch")

	// Late success routed into compensation exactly once.
	assert.Equal(t, 1, refunds)

	// Lifecycle untouched.
	lc, found := GetCache().GetLifecycle(ctx, tx)
	require.True(t, found)
	assert.Equal(t, SagaLifecycleFailed, lc)

	// Absorb was logged with the terminal skip reason.
	var absorbed bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonSagaTerminal {
			absorbed = true
		}
	}
	assert.True(t, absorbed)
}
```

(Imports: `cashshopmock "atlas-saga-orchestrator/cashshop/mock"`, `compartmentmock "atlas-saga-orchestrator/compartment/mock"`, `"time"`, tenant/uuid/logtest per file conventions. If the forward dispatch for `AwardAsset` uses a different compartment method, spy on `RequestCreateItemFunc` as well — both count toward `forward`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -race ./saga/ -run TestStepCompleted_TerminalAfterAccept -v`
Expected: FAIL — step advances / forward fires / no refund (any of the assertions).

- [ ] **Step 3: Implement the commit-time gate**

In `saga/processor.go`, `stepCompletedWithResultOnce`, insert immediately after the `GetById` error check (before the existing idempotency guard):

```go
	// Commit-time terminal gate (design §3.3a): AcceptEvent's fast-path
	// check can race the timeout transition. This guards the only function
	// that performs the forward write; the TryTransition version bump
	// (store.go) forces any concurrent optimistic writer back through here
	// via VersionConflictError retry. Outcome comes from the caller's
	// success flag — no kind table needed on this path.
	if lc, ok := GetCache().GetLifecycle(p.ctx, transactionId); ok && lc != SagaLifecyclePending {
		outcome := OutcomeFailure
		if success {
			outcome = OutcomeSuccess
		}
		step, stepOk := s.GetCurrentStep()
		p.absorbLateTerminal(s, lc, "step_completed", outcome, step, stepOk)
		return nil
	}
```

The existing `!success && !s.Failing()` TryTransition branch stays as-is (it also cancels the timer).

- [ ] **Step 4: Document the ordering invariant in `saga/lifecycle.go`**

Append to the package-level doc comment on `SagaLifecycleState` (PRD acceptance criterion):

```go
// Ordering invariant (task-135): a terminal TryTransition commit is a
// linearization point. Reads that begin after it (GetLifecycle, GetById)
// observe the terminal state; optimistic writes that began before it are
// invalidated by the TryTransition version bump (PostgresStore) or excluded
// by the cache mutex (InMemoryCache) and must re-read, at which point the
// stepCompletedWithResultOnce terminal gate absorbs the event. No code path
// writes the persisted status except TryTransition and the terminal-
// preserving Put/Remove in store.go.
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -race ./saga/ -run 'TestStepCompleted|TestAcceptEvent' -v && go test -race ./saga/... && go test -race -tags=test ./saga/...`
Expected: PASS, no regressions (existing duplicate-completion and failure-path tests unchanged).

- [ ] **Step 6: Commit**

```bash
git add saga/processor.go saga/lifecycle.go saga/processor_test.go
git commit -m "fix(saga-orchestrator): re-check lifecycle at step-completion commit time; document ordering invariant"
```

---

### Task 9: Deterministic task-102 race reproduction (integration, sqlite store)

**Files:**
- Test: `saga/late_event_integration_test.go` (new)

**Interfaces:**
- Consumes: everything above; `handleSagaTimeout` (`saga/timer.go:88`), `NewPostgresStore`/`SetCache`/`ResetCache` (`saga/cache.go`, `saga/store.go`), `SetEmitSagaFailedForTest` (Task 6).
- Produces: the PRD acceptance-criteria test — timeout → late `award_currency` success → (a) no forward advance, (b) exactly one inverse, (c) lifecycle stays failed, (d) exactly one Failed event, (e) redelivery no-op.

The production semantics under test (soft-deleted row still found by `GetById`) only exist on the Postgres store, so this test runs against `NewPostgresStore` backed by sqlite — not the in-memory cache (whose `Remove` hard-deletes).

- [ ] **Step 1: Write the failing integration test**

Create `saga/late_event_integration_test.go`:

```go
//go:build test

package saga

import (
	"context"
	"testing"
	"time"

	cashshopmock "atlas-saga-orchestrator/cashshop/mock"
	compartmentmock "atlas-saga-orchestrator/compartment/mock"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// TestLateEvent_TimeoutRacesCompletion reproduces the task-102 production
// sequence (PRD §1, design §4) deterministically: the timeout path runs to
// terminal first, then the in-flight award_currency_seller success arrives.
func TestLateEvent_TimeoutRacesCompletion(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Migration(db))
	SetCache(NewPostgresStore(db, logger))
	t.Cleanup(ResetCache)

	failedEvents := 0
	restore := SetEmitSagaFailedForTest(func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error {
		failedEvents++
		return nil
	})
	t.Cleanup(func() { SetEmitSagaFailedForTest(restore) })

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	// Two-step value-transfer saga: the seller credit is in flight at timeout.
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("mts-buy-test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 110}).
		AddStep("move_listing_to_holding", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 1, Item: ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	// 1. Timeout fires (invoked directly — no real timers): pending →
	//    compensating → failed → Remove (failed preserved) + one Failed event.
	handleSagaTimeout(logger, ctx, tx, 30*time.Second)
	lc, ok := GetCache().GetLifecycle(ctx, tx)
	require.True(t, ok)
	require.Equal(t, SagaLifecycleFailed, lc)
	require.Equal(t, 1, failedEvents)

	// 2. ~100ms later the seller-credit success arrives on the real processor path.
	refunds := 0
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(txId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
			refunds++
			assert.Equal(t, uint32(42), accountId)
			assert.Equal(t, int32(-110), amount, "seller's late payment must be clawed back")
			return nil
		},
	}
	forward := 0
	cp := &compartmentmock.ProcessorMock{
		RequestCreateItemFunc: func(uuid.UUID, uint32, uint32, uint32, time.Time) error { forward++; return nil },
		RequestCreateItemWithStatsFunc: func(uuid.UUID, uint32, uint32, uint32, time.Time, bool) error { forward++; return nil },
	}
	p := NewProcessor(logger, ctx).WithCashshopProcessor(cs).WithCompartmentProcessor(cp)

	_, accepted := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	assert.False(t, accepted, "(a) no forward progress")
	assert.Equal(t, 1, refunds, "(b) exactly one inverse dispatched")
	assert.Equal(t, 0, forward, "(a) next step never dispatched")

	lc, ok = GetCache().GetLifecycle(ctx, tx)
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleFailed, lc, "(c) saga stays terminal")
	assert.Equal(t, 1, failedEvents, "(d) exactly one Failed overall")

	var absorbed bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonSagaTerminal {
			absorbed = true
		}
	}
	assert.True(t, absorbed, "absorb must be logged with saga_terminal reason")

	// 3. Kafka at-least-once: the same event redelivered dispatches nothing (e).
	_, accepted = p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	assert.False(t, accepted)
	assert.Equal(t, 1, refunds, "(e) marker prevents double-compensation")
	assert.Equal(t, 1, failedEvents)
}

// TestLateEvent_FailureOutcomeAbsorbOnly: a late FAILURE report needs no
// rollback — the step's effect never landed (PRD §4.3).
func TestLateEvent_FailureOutcomeAbsorbOnly(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Migration(db))
	SetCache(NewPostgresStore(db, logger))
	t.Cleanup(ResetCache)

	restore := SetEmitSagaFailedForTest(func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error { return nil })
	t.Cleanup(func() { SetEmitSagaFailedForTest(restore) })

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 110}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))
	handleSagaTimeout(logger, ctx, tx, 30*time.Second)

	refunds := 0
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(uuid.UUID, uint32, uint32, int32) error { refunds++; return nil },
	}
	p := NewProcessor(logger, ctx).WithCashshopProcessor(cs)

	// cashshop has no failure kind for AwardCurrency in the acceptance table,
	// so exercise the generic path: a failure-classified kind that matches no
	// step absorbs without dispatch, and a StepCompleted(false) via the
	// commit-time gate also absorbs without dispatch.
	require.NoError(t, p.StepCompleted(tx, false))
	assert.Equal(t, 0, refunds, "failure outcome dispatches nothing")

	var absorbed bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonSagaTerminal {
			absorbed = true
		}
	}
	assert.True(t, absorbed)

	st, ok := GetCache().GetById(ctx, tx)
	require.True(t, ok)
	step, _ := st.StepAt(0)
	assert.False(t, step.LateCompensated(), "no claim on failure outcome")
	assert.Equal(t, Pending, step.Status(), "no step-status mutation")
}
```

- [ ] **Step 2: Run tests to verify current state**

Run: `go test -race -tags=test ./saga/ -run TestLateEvent -v`
Expected: PASS if Tasks 1–8 are complete (this test is the end-to-end proof; if any earlier task was skipped it fails loudly). If it fails, debug the specific assertion — do not weaken the test.

- [ ] **Step 3: Run every suite in the module**

Run: `go test -race ./... && go test -race -tags=test ./...`
Expected: PASS — including the pre-existing `createandequip`, `preset`, `step_event_matching`, `await_inventory_created` and `integration_test.go` suites (PRD: happy path unaffected).

- [ ] **Step 4: Commit**

```bash
git add saga/late_event_integration_test.go
git commit -m "test(saga-orchestrator): deterministic task-102 timeout-races-completion reproduction"
```

---

### Task 10: Full verification sweep

**Files:** none new — verification only (fix-and-rebuild cycles amend the relevant prior commit or add fix commits).

- [ ] **Step 1: Module verification**

From `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`:

```bash
go vet ./...
go build ./...
go test -race ./...
go test -race -tags=test ./...
```
Expected: all clean/PASS.

- [ ] **Step 2: Docker bake (mandatory per CLAUDE.md)**

From the worktree root:

```bash
docker buildx bake atlas-saga-orchestrator
```
Expected: image builds green. (go.mod was touched — Task 7's otel promotion — so this is not optional.)

- [ ] **Step 3: Redis key guard**

From the repo/worktree root:

```bash
tools/redis-key-guard.sh
```
Expected: clean (no new redis usage was added; this is the standing gate).

- [ ] **Step 4: Acceptance-criteria walkthrough**

Check each PRD §10 box against evidence (test name or file:line) and record the mapping in the task folder's notes or commit message:

- Terminal absorb before pending-step/action-mismatch checks → `TestAcceptEvent_TerminalLifecycleAbsorbs`.
- `SkipReasonSagaTerminal` log fields + metric → same test + span attributes in `absorbLateTerminal` (`saga/processor.go`).
- Late success compensated exactly once, idempotent → `TestCompensateLateStep_AwardCurrency_NegatedOnceOnly`, `TestLateEvent_TimeoutRacesCompletion` step 3.
- Late failure absorb-only → `TestLateEvent_FailureOutcomeAbsorbOnly`.
- Deterministic task-102 reproduction (a)–(c) + one Failed → `TestLateEvent_TimeoutRacesCompletion`.
- Existing suites unchanged → Step 1 output.
- Build/vet/test/bake → Steps 1–2.
- Ordering invariant documented → `saga/lifecycle.go` comment (Task 8).

- [ ] **Step 5: Commit any remaining fixes and stop**

Do NOT open a PR or invoke `superpowers:finishing-a-development-branch` — code review (`superpowers:requesting-code-review`) comes first per CLAUDE.md, and the ops follow-through (Tempo span-dimensions allowlist, see context.md) must be flagged in the PR description when it is opened.
