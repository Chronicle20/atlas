# Saga Terminal-State Race — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-08
---

## 1. Overview

The saga orchestrator runs each saga as an ordered list of steps, advancing on
Kafka status events keyed by transaction id, and enforcing a per-saga deadline.
When a saga misses its deadline, the timer transitions its lifecycle
`pending → compensating → failed`, dispatches compensation for the completed
steps, and emits one `Failed` event.

There is a race at the boundary. The lifecycle state machine
(`saga/lifecycle.go`) exists and is enforced by `Cache.TryTransition`, but the
event-acceptance gate `Processor.AcceptEvent` (`saga/processor.go`) does **not**
consult it. `AcceptEvent` only rejects on nil transaction id, saga-not-found,
no-pending-step, and step/event action mismatch. A saga that the timer has
already moved to `compensating`/`failed` still has `pending` steps in its
step-model, so a step-completion event that arrives *after* the timeout is
accepted and advances the saga **forward** — concurrently with the compensation
the timer already dispatched.

This was observed in production during task-102 MTS-marketplace testing
(transaction `3bd634ad`). A buy saga timed out; the timer compensated the
buyer's prepaid debit (refunding them), while ~100ms later the in-flight
`award_currency_seller` completion event progressed the same saga forward,
crediting the seller and moving the purchased item into the buyer's holding. Net
result: **the buyer received the item for free and the seller was paid** — the
currency/custody invariant was broken. This is not MTS-specific: any saga whose
steps involve slow cross-service round-trips can hit it, because the flaw is in
the shared orchestrator, not any one saga definition.

The correct model is that terminal lifecycle states are **absorbing**: once a
saga is compensating/failed/completed, no event may drive its forward step
walk. Crucially, a late step that genuinely *succeeded* has produced a real,
now-orphaned side effect (the seller credit, the item move). Absorbing the
event is necessary but not sufficient — that late success must itself be routed
into compensation so its effect is rolled back, restoring the invariant.

## 2. Goals

Primary goals:

- Make terminal saga lifecycle states (`compensating`, `failed`, `completed`)
  absorbing at `AcceptEvent`: a step-completion event for a saga in a terminal
  lifecycle state can never advance its forward step walk.
- When a late step-completion event reports **success** for a step whose effect
  is real (it ran a compensable action), route that step into compensation so
  its side effect is rolled back — no orphaned effects survive a timed-out
  saga.
- Emit a distinct, structured, queryable signal (log fields + metric) whenever a
  late-after-terminal event is absorbed, so these races are observable and
  post-incident-auditable.
- Prove the fix with a deterministic test that reproduces the timeout-races-
  completion ordering and asserts no forward progress + effect rollback.

Non-goals:

- Repairing sagas already left inconsistent by the pre-existing bug (a one-off
  data-reconciliation exercise — tracked separately if needed).
- Changing any saga's timeout duration or the per-step budget (the MTS buy/list
  budgets were already tuned in task-102, commit a655d0654f).
- Introducing a grace window after the deadline — the deadline is a hard cutoff
  (see §8, decision confirmed at spec time).
- A dead-letter topic for rejected events — structured logging + a metric is the
  agreed observability surface.
- Changing the compensation dispatch mechanics themselves (reverse-walk,
  per-action compensators) beyond adding the late-success entry point.

## 3. User Stories

- As a **marketplace buyer**, when my purchase saga times out, I want the
  outcome to be all-or-nothing — either I paid and received the item, or I was
  refunded and received nothing — so that a slow broker never hands out free
  items or double-charges.
- As a **saga-owning service author**, I want the orchestrator to guarantee that
  a timed-out saga cannot both compensate and continue, so I don't have to guard
  every step handler against terminal-state races myself.
- As an **operator**, I want a distinct log + metric when a late event is
  absorbed after a saga went terminal, so I can see how often this race fires,
  correlate it with broker latency, and audit any effects that were rolled back.
- As an **on-call engineer**, when the currency/custody invariant is at risk, I
  want the orchestrator to self-heal the late-success effect via compensation
  rather than require manual DB surgery.

## 4. Functional Requirements

### 4.1 Lifecycle gate at event acceptance

- `AcceptEvent(transactionId, kind)` MUST reject the event when the saga's
  current lifecycle state (from `Cache.GetLifecycle` / equivalent) is any
  terminal state: `compensating`, `failed`, or `completed`.
- The rejection MUST occur before the no-pending-step and action-mismatch checks
  so a terminal saga with residual pending steps cannot be advanced.
- Rejection returns the same `(AcceptDecision{}, false)` contract as the existing
  skip paths — the caller does no forward work.
- A saga in `pending` lifecycle continues to be accepted exactly as today (no
  behavior change on the happy path).

### 4.2 Distinct skip reason + observability

- Add a new skip reason constant (e.g. `SkipReasonSagaTerminal`) distinct from
  the existing reasons, logged via the existing `LogSkip` structured path with
  fields: `transaction_id`, `event_kind`, `lifecycle_state`, `step_id` (if a
  step would otherwise have matched).
- Increment a counter metric (e.g. `saga_late_event_absorbed_total`) labeled by
  `saga_type` and `lifecycle_state`, so the rate is graphable.

### 4.3 Late-success compensation routing

- When an absorbed event reports a step **success** (not failure) for a step
  whose action is compensable, the orchestrator MUST enqueue that step's
  compensation so its real side effect is rolled back.
  - The observed case: a `award_currency_seller` (or `mts_move_listing_to_holding`)
    step completes after the timeout. Its effect (points credited / item moved)
    must be reversed by the corresponding compensating action, the same one the
    reverse-walk would have used had the step been part of the compensated set.
- An absorbed event reporting **failure** requires no compensation (the step's
  effect never landed) — absorb and log only.
- Late-success compensation MUST be idempotent and safe under duplicate delivery:
  the same late event redelivered (Kafka at-least-once) must not double-compensate.
  Reuse the existing per-step / per-transaction idempotency guards; a step already
  compensated is a no-op.
- Late-success compensation MUST NOT re-transition the lifecycle out of its
  terminal state (it stays `failed`); it only dispatches the missing rollback and
  records that it did so.

### 4.4 Ordering and concurrency

- The lifecycle transition to a terminal state and the acceptance gate MUST be
  observed consistently: once `TryTransition(... → compensating/failed)` commits,
  every subsequent `AcceptEvent` for that transaction id sees the terminal state
  (no read-before-write gap that re-opens the race). Use the cache's existing
  optimistic-version / transition primitives; document the ordering invariant.
- Concurrent delivery of the timeout and the step-completion event must resolve
  to exactly one outcome: terminal + (if the late step succeeded) its
  compensation dispatched. Neither a forward advance nor a double-compensation is
  permitted.

### 4.5 Hard deadline (no grace)

- The deadline remains a hard cutoff. A step completion that lands after the
  deadline is treated as late (absorbed + compensated if it succeeded), never as
  a success that revives the saga. No grace/extension window is introduced.

## 5. API Surface

No external REST or Kafka contract changes.

- Internal: `AcceptEvent` gains a lifecycle check (signature unchanged). A new
  internal entry point (e.g. `CompensateLateStep(transactionId, step, result)`)
  is added on the processor/compensator for §4.3; it is not exposed over REST or
  Kafka.
- Existing status events (`EVENT_TOPIC_SAGA_STATUS`, the per-domain status
  topics) are unchanged. No new topic. The already-emitted single `Failed` event
  on timeout is unchanged.

## 6. Data Model

No new persistent entities. The saga cache entry already tracks lifecycle state
(`saga/lifecycle.go`, `saga/cache.go`) and per-step status. This task may add:

- A per-step marker that a late-success compensation was dispatched (to keep
  §4.3 idempotent), stored on the existing in-memory/cache step model — no schema
  migration if the orchestrator's saga store is not relational for this field;
  confirm during design whether the saga store persists steps and, if so, whether
  the marker needs a column. Multi-tenancy: all lookups remain scoped by the
  saga's existing `tenant_id`; no cross-tenant access is introduced.

## 7. Service Impact

- **atlas-saga-orchestrator** — the only service that changes:
  - `saga/processor.go` `AcceptEvent`: add the terminal-lifecycle gate.
  - `saga/lifecycle.go` / `saga/cache.go`: expose a lifecycle read used by the
    gate (if not already available) and document the transition/observe ordering.
  - `saga/compensator.go`: add the late-success single-step compensation entry
    point and its idempotency guard.
  - Skip-reason constants + metric registration.
  - Tests: `saga/*_test.go` and an integration test reproducing the race.
- **All saga-initiating services** (atlas-mts, atlas-character-factory,
  cash-shop, etc.): no code change; they inherit the guarantee. A regression that
  reintroduced forward progress after terminal would surface in their integration
  suites.

## 8. Non-Functional Requirements

- **Correctness (primary):** after this task, no saga can both compensate and
  advance forward; no late-successful step leaves an orphaned effect. The
  currency/custody invariant that failed in task-102 holds under the
  timeout-races-completion ordering.
- **Idempotency:** all new compensation dispatch is safe under Kafka
  at-least-once redelivery.
- **Observability:** the absorb path is logged with structured fields and a
  labeled metric; the rate is graphable per saga type.
- **Performance:** the gate is an O(1) lifecycle read on the hot event path; no
  measurable added latency on the happy path.
- **Determinism in tests:** the race is exercised by a test that controls event
  ordering (inject the completion after the timeout transition), not by timing.
- **Multi-tenancy:** all new lookups remain tenant-scoped via the saga's
  existing tenant context.
- **Decision (confirmed at spec time):** late-successful steps are routed into
  compensation (option B, not merely ignored); the deadline is a hard cutoff (no
  grace window); observability is structured log + metric (no dead-letter topic).

## 9. Open Questions

- Does the orchestrator's saga store persist per-step status durably, or is the
  cache authoritative and rebuilt on restart? This decides whether the §4.3
  idempotency marker needs a migration or is purely in-cache. (Resolve in
  design by reading `saga/store.go` / `saga/entity.go`.)
- Are all step actions that can land late actually compensable (do they all have
  a registered compensator)? Enumerate the actions that appear as non-final
  steps and confirm each has a reverse action; flag any that don't as a gap.
- Is `Cache.GetLifecycle` already exposed and race-safe for a read on the event
  path, or does the gate need a new accessor with the same versioning guarantees
  as `TryTransition`?

## 10. Acceptance Criteria

- [ ] `AcceptEvent` rejects step-completion events for sagas in `compensating`,
      `failed`, or `completed` lifecycle states, before the pending-step /
      action-mismatch checks, returning the standard skip contract.
- [ ] A distinct `SkipReasonSagaTerminal` (or equivalent) is logged with
      `transaction_id`, `event_kind`, `lifecycle_state`, and a labeled metric is
      incremented.
- [ ] A late step-completion event reporting **success** for a compensable step
      dispatches that step's compensation exactly once (idempotent under
      redelivery); the saga stays terminal and its late effect is rolled back.
- [ ] A late step-completion event reporting **failure** is absorbed and logged
      with no compensation dispatched.
- [ ] A deterministic test reproduces the task-102 ordering (timeout →
      compensation dispatched → late `award_currency_*` success) and asserts:
      (a) no forward step advance, (b) the late effect is compensated, (c) exactly
      one `Failed` event overall.
- [ ] Existing saga integration suites (character-creation, preset, MTS custody)
      pass unchanged — the happy path is unaffected.
- [ ] `go build`, `go vet`, `go test -race ./...` clean in
      atlas-saga-orchestrator; `docker buildx bake atlas-saga-orchestrator` green.
- [ ] The ordering invariant (terminal transition is observed by all subsequent
      `AcceptEvent` calls) is documented in `saga/lifecycle.go`.
