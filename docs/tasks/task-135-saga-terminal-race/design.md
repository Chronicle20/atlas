# Saga Terminal-State Race — Design

Task: task-135-saga-terminal-race
Status: Approved PRD → design
Inputs: `docs/tasks/task-135-saga-terminal-race/prd.md`
Service: `services/atlas-saga-orchestrator` (only service changed)

---

## 1. Code reality (what exists today)

All paths below are relative to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`.

- **Lifecycle machine exists and is enforced at emission points, not acceptance.**
  `saga/lifecycle.go` defines `pending → compensating → failed` and `pending →
  completed`; `Cache.TryTransition` enforces it. The timer
  (`saga/timer.go:handleSagaTimeout`), the failure branch of
  `stepCompletedWithResultOnce` (`saga/processor.go:443`), sync-error emission
  (`processor.go:emitFailedFromStepSyncError`), and completion
  (`processor.go:886`) all take the guard. `AcceptEvent`
  (`saga/processor.go:362`) does **not** — it checks nil-tx, saga-not-found,
  no-pending-step, action-mismatch only.

- **`Cache.GetLifecycle` already exists on both implementations** (PRD open
  question 3 — resolved: no new accessor needed). `InMemoryCache` reads it
  under the same `RWMutex` that `TryTransition` writes under
  (`saga/cache.go:210`). `PostgresStore` reads the `status` column
  (`saga/store.go:314`); `TryTransition` is an atomic conditional `UPDATE`
  (`store.go:290`), so any read that starts after the commit observes it.

- **The production store is Postgres; the saga (steps included) is one JSONB
  blob.** `main.go:76` wires `saga.NewPostgresStore` via `saga.SetCache`.
  `Entity.SagaData` (`saga/entity.go`) holds the full serialized `Saga`
  including per-step status (PRD open question 1 — resolved: a new per-step
  marker is a JSON field inside `SagaData`; **no schema migration**).
  `InMemoryCache` is effectively test-only.

- **Two timeout paths lead to terminal state**: the in-process
  `TimerRegistry` (`saga/timer.go`) and the DB reaper
  (`main.go:reapTimedOutSagas`, `SELECT ... FOR UPDATE SKIP LOCKED` over
  `timeout_at < now()`). Both end with the saga's lifecycle terminal and
  compensation dispatched (reverse-walk for CharacterCreation/PetEvolution,
  per-step compensators otherwise).

- **Three defects widen the race beyond the PRD's ~100ms framing:**

  1. **The post-eviction window is unbounded, not transient.**
     `PostgresStore.Remove` is a *soft delete* (`store.go:193` sets
     `status='completed'`), but `GetById` (`store.go:73`) does **not** filter
     by status. A step-completion event arriving minutes after the timeout
     still finds the saga, still sees pending steps, and still advances it.
     The race is not confined to the dispatch window between
     `TryTransition` and `Remove`.

  2. **`TryTransition` does not bump the optimistic-lock `version`**
     (`store.go:290` updates `status` only), and **`Put` recomputes `status`
     from `saga.Failing()`** (`store.go:110`). Interleaving:
     `AcceptEvent` reads lifecycle `pending` → timer commits
     `pending→compensating` → the handler's `StepCompleted(true)` does
     `MarkEarliestPendingStepWithResult` → `Put` succeeds (version unchanged
     by the transition) and **overwrites `status` back to `'active'`**,
     re-opening the saga. Gating `AcceptEvent` alone does not close this.

  3. **The timer erases the `failed` audit state.** `handleSagaTimeout` runs
     `TryTransition(compensating→failed)` and then `c.Remove(...)`, which
     (on Postgres) overwrites `status='failed'` with `'completed'`. A
     timed-out saga's row ends up indistinguishable from a successful one.

## 2. Approaches considered

**A. Gate `AcceptEvent` only (PRD letter, minimal).** Add the lifecycle check
and skip-reason; nothing else. Rejected as incomplete: defect 2 means the
timeout can commit *between* the gate's read and the handler's
`StepCompleted` write, and the write then resurrects the saga
(`status='active'`). The PRD's §4.4 ordering requirement ("no
read-before-write gap that re-opens the race") is unsatisfiable without a
commit-time guard.

**B. Gate at `AcceptEvent` + commit-time enforcement + late-success routing
(recommended).** Layered:
   - fast-path gate in `AcceptEvent` (absorb + observe + route),
   - authoritative re-check in `stepCompletedWithResultOnce` (the only place
     that knows `success` and performs the forward write),
   - `TryTransition` bumps `version` so every in-flight optimistic write that
     predates the terminal commit fails with `VersionConflictError` and
     re-checks on retry,
   - `Put`/`Remove` made terminal-preserving so no code path can regress a
     terminal status.
   This makes terminal states absorbing *by storage-level invariant*, not by
   convention at one call site.

**C. Per-saga serialized event loop (actor model).** Route every event,
timer fire, and REST mutation for a transaction id through one goroutine;
races become structurally impossible. Rejected: a large re-architecture of
the consumer topology and the DB-reaper/multi-replica story for a defect that
layered guards close deterministically. Revisit only if the orchestrator
accretes more concurrent mutators.

**Late-success routing placement** (within B):
   - *(i) handler-level* — teach every consumer to branch on a
     "terminal-skip" result and call a compensation entry point. Rejected:
     ~30 call sites of churn (`kafka/consumer/*/consumer.go` all use the
     identical `if _, ok := p.AcceptEvent(...); !ok { return }` idiom), and
     each handler would need duplicate success/failure logic.
   - *(ii) inside `AcceptEvent` via an EventKind outcome classification
     (chosen)* — the kind itself encodes outcome (`asset.created` = success
     signal; `compartment.error`, `character.creation_failed`,
     `character.meso_error`, `storage.error`, … = failure signal). One new
     table in `saga/event_acceptance.go`, mirroring the existing
     `acceptanceTable` + completeness-test pattern. Handlers stay untouched
     (`(AcceptDecision{}, false)` contract preserved, per PRD §4.1/§5).
   - *(iii) only in `stepCompletedWithResultOnce`* — insufficient alone: once
     `AcceptEvent` rejects, the handler returns and `StepCompleted` is never
     called, so the common post-terminal delivery would never route.
     It is retained as the *second* gate for the TOCTOU interleave.

## 3. Design

### 3.1 EventKind outcome classification (`saga/event_acceptance.go`)

New table + accessor, same shape and test discipline as `acceptanceTable`:

```go
type EventOutcome string

const (
    OutcomeSuccess EventOutcome = "success"
    OutcomeFailure EventOutcome = "failure"
)

// outcomeTable classifies each EventKind as a success signal (the step's
// side effect landed) or a failure signal (it did not). Late-after-terminal
// routing uses this to decide whether a rollback must be dispatched.
var outcomeTable = map[EventKind]EventOutcome{ ... }
```

Failure kinds: `character.creation_failed`, `character.meso_error`,
`compartment.creation_failed`, `compartment.error`,
`inventory.creation_failed`, `storage.error`, `storage.compartment_error`,
`cashshop.compartment_error`, `invite.rejected`. Every other kind currently
in `acceptanceTable` is a success signal. A new completeness test in
`event_acceptance_test.go` asserts every kind referenced anywhere in
`acceptanceTable` has an `outcomeTable` entry (same guard style as the
existing coverage test), so adding a kind without classifying it fails CI.

`invite.rejected` is classified `failure` deliberately: a rejected invite
left no side effect to roll back; the handler-level semantics (reject → step
failed) agree.

New skip reason: `SkipReasonSagaTerminal = "saga_terminal"` alongside the
existing constants.

### 3.2 The gate in `AcceptEvent` (`saga/processor.go`)

Placement: after the saga-not-found check (the saga model is needed for step
matching and logging), **before** the no-pending-step and action-mismatch
checks (PRD §4.1).

```
s, err := p.GetById(transactionId)            // existing
if err != nil { ... saga_not_found ... }      // existing

if lc, ok := GetCache().GetLifecycle(p.ctx, transactionId); ok && lc != SagaLifecyclePending {
    p.absorbLateTerminalEvent(s, lc, kind)    // log + span + maybe compensate
    return AcceptDecision{}, false
}
```

`absorbLateTerminalEvent(s Saga, lc SagaLifecycleState, kind EventKind)`:

1. Match the would-be step exactly as the happy path does: earliest pending
   step (`s.GetCurrentStep()`) + `StepAcceptsEvent(step.Action(), kind)`.
   Steps are dispatched serially (a step's command goes out only when the
   previous step's completion event arrives), so at most one step is in
   flight at timeout — the earliest pending step *is* the in-flight one.
2. `LogSkip` with reason `saga_terminal` and fields `transaction_id`,
   `event_kind`, `lifecycle_state`, `saga_type`, plus `step_id` when a step
   matched (PRD §4.2).
3. Emit the observability span (§3.6).
4. If a step matched **and** `outcomeTable[kind] == OutcomeSuccess`: call
   `p.comp.CompensateLateStep(s, step)` (§3.4). Failure outcome, or no
   matching step: absorb-only.

The nil-tx path is untouched; a `pending` lifecycle (or a cache miss on
`GetLifecycle`, which can only happen when `GetById` raced a hard delete in
the in-memory impl) falls through to the existing checks — zero happy-path
behavior change (PRD §4.1).

### 3.3 Commit-time enforcement (closing PRD §4.4)

Three coordinated changes make the terminal transition impossible to miss
*and* impossible to overwrite:

**(a) `stepCompletedWithResultOnce` re-checks lifecycle at the top.** After
its `GetById`, if the lifecycle is terminal: route through the same
`absorbLateTerminalEvent` core (here `success` is a parameter, so the
outcome table is not needed — the shared internal takes an explicit outcome)
and return nil. This is the authoritative gate: it guards the only function
that performs the forward write, and it catches the interleave where the
timeout commits after `AcceptEvent` passed. The existing
`!success` branch's `TryTransition` guard (`processor.go:443`) is subsumed
but retained — it also cancels the timer, which stays.

**(b) `PostgresStore.TryTransition` bumps `version`** (`SET status = ?,
version = version + 1, updated_at = ?`) and invalidates the instance's
`ver`-map entry for that transaction id. Consequence: every optimistic `Put`
built on a pre-terminal read — on this replica or any other — fails with
`VersionConflictError`. The retry loop in `StepCompletedWithResult` re-runs
`stepCompletedWithResultOnce`, whose new top-of-function check (a) absorbs.
`InMemoryCache` needs no change: its single mutex means any `Put` after the
transition already observes the terminal lifecycle via check (a).

**(c) `PostgresStore.Put` and `Remove` become terminal-preserving.**
- `Put`: the status assignment becomes a SQL `CASE` — status is written from
  `saga.Failing()` only when the current row status is `active` or
  `compensating`; `failed`/`completed` are preserved
  (`status = CASE WHEN status IN ('failed','completed') THEN status ELSE ? END`).
  This is belt-and-braces behind (b): even a `Put` built on a *fresh* read of
  a terminal saga (e.g. an `AddStep` retry that raced the timeout) cannot
  resurrect it. `SagaData` may still be updated (needed for the §3.5 marker).
- `Remove`: same `CASE` — `failed` stays `failed`; only
  `active`/`compensating`/`completed` collapse to `completed`. This fixes
  defect 3 (timed-out sagas no longer masquerade as completed in the DB) and
  makes the `lifecycle_state` observability label truthful. Both expressions
  are portable SQL (`gorm.Expr`), compatible with the sqlite test dialect.

**Ordering invariant, documented in `saga/lifecycle.go` (PRD acceptance
criterion):** *a terminal `TryTransition` commit is a linearization point.
Reads that begin after it (`GetLifecycle`, `GetById`) observe the terminal
state; writes that began before it are invalidated by the version bump and
must re-read, at which point the `stepCompletedWithResultOnce` gate absorbs.
No code path may write `status` except through `TryTransition` or the
terminal-preserving `Put`/`Remove`.*

### 3.4 `CompensateLateStep` (`saga/compensator.go`)

New interface method:

```go
// CompensateLateStep dispatches the single-step inverse for a step whose
// success event arrived after the saga went terminal. Pure dispatch — no
// lifecycle transitions, no Failed emission, no cache eviction (the saga is
// already terminal and stays terminal; PRD §4.3).
CompensateLateStep(s Saga, step Step[any]) error
```

Implementation: a per-action switch that dispatches the inverse computed
from the **step payload** (never the event payload), reusing the exact
inverse idioms already proven in the reverse-walks
(`DispatchCharacterCreationRollbacks`, `DispatchPetEvolutionRollbacks`) and
the per-step compensators:

| Late-completed action | Inverse dispatched | Existing idiom reused |
|---|---|---|
| `AwardAsset` | `compP.RequestDestroyItem` | char-creation reverse-walk |
| `CreateAndEquipAsset` | `compP.RequestDestroyItem` | char-creation reverse-walk |
| `CreateSkill` | `skillP.RequestDeleteSkill` | char-creation reverse-walk |
| `CreateCharacter`, `AwaitCharacterCreated` | `charP.RequestDeleteCharacter` | char-creation reverse-walk |
| `DestroyAsset`, `DestroyAssetFromSlot` | `compP.RequestCreateItem` | pet-evolution reverse-walk |
| `AwardMesos` | `charP.AwardMesosAndEmit` with `-Amount` | pet-evolution reverse-walk |
| `AwardCurrency` | wallet credit with negated amounts | negation, same pattern as mesos |
| `AwardExperience` / `DeductExperience` | experience change with negated amount | negation |
| `AwardFame` | fame change with negated amount | negation |
| `EquipAsset` | unequip (swap source/destination) | `compensateEquipAsset` |
| `UnequipAsset` | equip (swap back) | `compensateUnequipAsset` |

**Not compensable in v1 — absorb + WARN (`late_effect_unrecoverable`) +
span attribute `late.compensated=false`:** cosmetic changes
(`ChangeHair`/`Face`/`Skin` — payload lacks the prior value; same limitation
already documented on their step-failure compensators), `ChangeJob`, `SetHP`,
`ResetStats`, `RebalanceAP`, `CancelAllBuffs`, quest ops
(`StartQuest`/`CompleteQuest`/`SetQuestProgress`/`ForfeitQuest`), skill
`UpdateSkill`, `IncreaseBuddyCapacity`, `GainCloseness`, `EvolvePet`,
consumable effects, guild ops, `CreateInvite`, storage/cash-shop compartment
moves (`AcceptTo*`/`ReleaseFrom*` — multi-phase custody transfers whose safe
inverse depends on which phase landed), `AwaitInventoryCreated`.

This answers PRD open question 2: **no, not every event-completable action
has a registered inverse today.** The v1 compensable set covers the entire
value-transfer class that broke the task-102 invariant (currency, items,
mesos, experience, character/skill creation). The remainder are explicitly
enumerated, logged loudly when hit, and countable via the metric — the data
to justify widening the set later. Actions with empty `acceptanceTable`
entries (fire-and-forget warps, messages, PQ ops, …) can never produce a
late event and need no entry.

### 3.5 Idempotency marker (PRD §4.3, open question 1)

New private field on `Step[T]`: `lateCompensated bool`, serialized through
the existing step JSON DTO (`latecompensated` key in `SagaData`; JSONB blob,
no migration). Accessors follow the model conventions
(`LateCompensated()`, plus a `Saga.WithStepLateCompensated(index)` copy-on-
write helper mirroring `WithStepStatus`).

`CompensateLateStep` is **claim-then-dispatch**:

1. Under `AtomicUpdateSaga` (existing optimistic-version retry helper), check
   `step.LateCompensated()`; if set → return nil (duplicate delivery, no-op).
   Otherwise set it and persist.
2. Only the goroutine whose `Put` won the version race proceeds to dispatch
   the inverse command. Losers see the marker on re-read.

Ordering tradeoff (documented in code): claiming before dispatching gives
**at-most-once** rollback. The alternative (dispatch-then-mark, at-least-
once) is unsafe here because the negation inverses (`AwardMesos`,
`AwardCurrency`, experience, fame) are not idempotent downstream — a
double-dispatch would double-refund and corrupt balances the other way. A
crash in the window between claim and dispatch loses the rollback, but the
structured log + span emitted before the claim make it auditable, which
matches the PRD's observability-over-dead-letter decision. Step status is
deliberately **not** changed (no `Failed` marking): the saga is terminal, and
mutating step status could interfere with restart recovery of
`compensating`-status sagas (`main.go:recoverSagas` re-drives `Step`).

The lifecycle is never re-transitioned (PRD §4.3): `CompensateLateStep`
performs no `TryTransition`, and the terminal-preserving `Put` (§3.3c) means
even its marker write cannot alter `status`.

### 3.6 Observability (PRD §4.2)

- **Structured log** (primary, Loki-queryable): the `LogSkip` call in §3.2
  with reason `saga_terminal`; plus a WARN with reason
  `late_effect_unrecoverable` for the non-compensable-success case, and an
  INFO `late_effect_compensated` after a successful claim+dispatch.
- **Metric**: Atlas has no in-process Prometheus registry; the established
  pipeline (task-040) is Tempo `metrics_generator` span-metrics. Emit a
  dedicated span `saga.late_event_absorbed` (child of the consumer span via
  the ambient context tracer) with attributes `saga.type`,
  `saga.lifecycle_state`, `late.outcome` (`success`/`failure`),
  `late.compensated` (`true`/`false`). The counter is then
  `traces_spanmetrics_calls_total{span_name="saga.late_event_absorbed"}`
  grouped by those dimensions — the PRD's
  `saga_late_event_absorbed_total` under the repo's existing metric scheme.
  Cardinality: ~10 saga types × 3 lifecycle states × 2 × 2 — trivially within
  the task-040 budget; `transaction.id` is on the forbidden-attribute list
  and stays out of the span (it's in the log line instead).
  Out-of-tree follow-through (same runbook as task-040): append `saga.type`,
  `saga.lifecycle_state`, `late.outcome`, `late.compensated` to the Tempo
  `span_metrics.dimensions` allowlist ConfigMap; Tempo hot-reloads.
  Alternative rejected: introducing `client_golang` + a scrape target for one
  counter — new cluster infrastructure duplicating an existing pipeline.

### 3.7 Multi-tenancy

No new lookup shapes: the gate and routing use the consumer's existing
tenant-scoped context (`tenant.MustFromContext` inside cache/store, tenant
header parsed by the consumer, timer re-wraps its tenant). The span carries
`tenant.id` like task-040 spans. No cross-tenant access is introduced.

## 4. Failure-ordering walkthrough (the task-102 sequence, post-fix)

1. Buy saga times out → timer wins `TryTransition(pending→compensating)`;
   version bumps; reverse dispatches go out; `compensating→failed`; `Remove`
   preserves `failed`.
2. ~100ms later `cashshop.wallet_updated` (the seller credit) arrives with
   the saga's transaction id. `AcceptEvent` → `GetById` finds the row
   (soft-deleted, unfiltered — unchanged) → `GetLifecycle` = `failed` →
   absorb: log `saga_terminal`, span, outcome=`success`, matched step
   `award_currency_seller` → `CompensateLateStep` claims the marker and
   dispatches the negated wallet credit. Seller's late payment is clawed
   back; buyer was already refunded by the timer. Invariant holds.
3. Kafka redelivers the same event → gate absorbs again → marker already set
   → no second dispatch.
4. Alternative interleave: the event passes `AcceptEvent` *before* the timer
   commits. The handler's `StepCompleted(true)` →
   `stepCompletedWithResultOnce` either (i) sees terminal at its top check →
   absorbs + routes, or (ii) raced past it, and its `Put` fails on the bumped
   version → retry → check (i) absorbs. No forward advance is reachable, and
   exactly one `Failed` was emitted (by the timer; nothing here emits).

## 5. Testing (PRD §8 determinism, acceptance criteria)

All in `services/atlas-saga-orchestrator` (package `saga` unless noted),
using the existing builder/mocks/`ResetCache` seams — no timing-based tests.

1. **Gate unit tests** (`accept_event_test.go`): for each terminal state
   (`compensating`, `failed`, `completed`) `AcceptEvent` returns
   `(AcceptDecision{}, false)` even with a pending, action-matching step;
   `pending` still accepts; the skip is logged with reason `saga_terminal`
   (log-hook assertion, existing pattern).
2. **Outcome-table completeness** (`event_acceptance_test.go`): every
   `EventKind` referenced in `acceptanceTable` is classified.
3. **Deterministic race reproduction** (new
   `late_event_integration_test.go`): a two-step value-transfer saga; invoke
   the timeout path directly (`handleSagaTimeout` — no real timers), then
   deliver the late success through the real processor path. Assert:
   (a) no forward handler dispatch (handler spy),
   (b) exactly one inverse dispatch (processor mock),
   (c) lifecycle stays `failed`,
   (d) exactly one `Failed` event (producer test-seam),
   (e) re-delivering the same event dispatches nothing (marker).
4. **TOCTOU interleave**: accept first, transition terminal, then call
   `StepCompleted(true)` — assert absorb-and-route, no `MarkEarliestPending…`
   forward write, no status regression.
5. **Late failure event**: absorbed, logged, zero compensation dispatched.
6. **Non-compensable late success**: absorbed, `late_effect_unrecoverable`
   WARN, no dispatch.
7. **Store guards** (`store_test.go`, gorm + sqlite, new): `TryTransition`
   bumps `version` (stale `Put` → `VersionConflictError`); `Put` cannot
   regress `failed`→`active`; `Remove` preserves `failed`.
8. **Marker round-trip** (`model_test.go`): `lateCompensated` survives
   `MarshalJSON`/`UnmarshalJSON` (guards the JSONB persistence).
9. **Regression**: existing suites (`createandequip`, `preset`,
   `step_event_matching`, `integration_test.go`) unchanged and green —
   happy-path behavior is untouched by construction (gate no-ops on
   `pending`).

Verification per CLAUDE.md: `go test -race ./...`, `go vet ./...`,
`go build ./...` in the module; `docker buildx bake atlas-saga-orchestrator`;
`tools/redis-key-guard.sh`.

## 6. Out of scope (per PRD non-goals)

- Reconciling sagas already corrupted before this fix.
- Timeout durations / per-step budgets (task-102 tuned them).
- Grace windows (hard cutoff confirmed), dead-letter topics.
- Widening the compensable set beyond the value-transfer class (the metric
  and WARN logs will show whether any non-compensable action is ever hit in
  practice).
- The `CompensateFailedStep` default-case "reset step to Pending" retry
  semantics (`compensator.go:259`) — pre-existing, orthogonal, and only
  reachable pre-terminal.

## 7. Open questions — resolved

| PRD §9 question | Answer |
|---|---|
| Is per-step status durable or cache-rebuilt? | Durable: the whole `Saga` (steps included) is one JSONB `SagaData` blob in Postgres; the marker is a step JSON field, no migration (§3.5). |
| Are all late-landing actions compensable? | No. §3.4 enumerates the v1 compensable set (full value-transfer class) and the absorb-only remainder, each logged as `late_effect_unrecoverable` when hit. |
| Is `GetLifecycle` race-safe for the gate? | Yes on both impls (§1), but a read-side gate alone is insufficient — the version bump + `stepCompletedWithResultOnce` re-check + terminal-preserving writes close the write-side race (§3.3). |
