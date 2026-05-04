# Use-Item Server Latency Cleanup — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-30
---

## 1. Overview

A real Tempo trace of a single HP-potion use (`itemId 2000013`, character 12, trace `665342ad8db44ec8ce4b5bb108e5de7e`) showed the server holding the request open for ~368ms of visible activity (~700ms of total root span). Three independent contributors stood out: redundant party-announce work on solo characters logged as an error, saga-orchestrator running saga lookups on every event-bus message including ones with nil transaction IDs, and three independent REST calls run sequentially in the consumables consume path that could be issued concurrently.

Each issue is small in isolation; bundling them lets us iterate on a single trace as a measurement target. The user-perceived motivation is the "feels exaggerated" lag between potion use and next attack — although the in-game animation lock dominates that gap (~500ms client-side), shrinking the server budget gives more headroom and stops misleading log noise from showing up in routine play.

## 2. Goals

Primary goals:
- Stop logging non-error states as errors. Specifically, "Unable to announce character health to party members" must not fire for characters who are not in a party.
- Adopt the existing decorator pattern for party data on `atlas-channel`'s character model, so consumers that need party state get it via `cp.GetById(cp.PartyDecorator)(...)` rather than ad-hoc party-processor calls.
- Eliminate gratuitous saga-table lookups for events whose `transactionId` is the zero UUID, since those events cannot belong to a saga by definition.
- Run the three independent reads in `consumable.ConsumeStandard` (and structurally similar `Consume*` variants where the same pattern holds) concurrently rather than sequentially, using the project's existing `model.ParallelMap` helper where it fits or `errgroup` where it does not.

Non-goals:
- Designing a "saga-relevant" filter at the Kafka consumer level (e.g., header-based or topic-split filtering of non-saga events). Defer to a future task.
- Adding `partyId` to atlas-character's REST contract or storage. The decorator on atlas-channel fetches party state from atlas-parties; atlas-character is not modified.
- Caching party state in atlas-channel to avoid the REST call. Out of scope; same REST-call count as today.
- Removing the `GET /characters/{id}` at the start of `handleStatusEventStatChanged` — the event payload does not carry HP/MaxHp/MapId, so the GET is load-bearing.
- Parallelizing `ConsumeScroll` or any consume path whose reads have inter-dependencies; only paths whose initial reads are demonstrably independent are in scope.
- Adding business-attribute span enrichment (e.g., `character.id` on spans). Separate observability task.
- Changing saga subscription topology or the `acceptanceTable` model.
- Frontend / atlas-ui changes. None expected.

## 3. User Stories

- As an operator reading logs while a player solos, I want to **not** see "Unable to announce character health to party members" debug-with-error lines, so my log volume reflects real failures.
- As a developer reading `handleStatusEventStatChanged`, I want party data to come through the same `cp.GetById(cp.PartyDecorator)` decorator pattern as inventory, pets, skills, and quests, so the code is consistent across the codebase.
- As a developer reading saga-orchestrator consumer code, I want non-saga events to short-circuit before any DB lookup when the `transactionId` is the zero UUID, so the saga subsystem does no work for events that cannot possibly belong to a saga.
- As a player using a potion, I want the server's processing budget to be tight enough that the next-attack cooldown is no longer dominated by server-side delay.
- As a developer iterating on this fix, I want a reproducible Tempo trace measurement so we can verify each step of the cleanup landed.

## 4. Functional Requirements

### 4.1 atlas-channel — Party Decorator and Announce Cleanup

- Define a `party.Model` shape (or reuse the existing one in `atlas-channel/party`) suitable for attaching to the character model.
- Add an optional `party *party.Model` (or equivalent reference; pointer permitted to express "not loaded" vs "loaded but absent" if needed) to `atlas-channel/character/model.Model`.
- Add corresponding builder support (mirroring the `InventoryDecorator` shape) so the field is populated only when explicitly requested.
- Add `PartyDecorator(m Model) Model` to `character.ProcessorImpl`, sourcing party state via the existing `party.NewProcessor.ByMemberIdProvider`. When the character is not in a party, the decorator must complete successfully with the party field representing "not in a party"; this MUST NOT be returned as an error.
- Add accessor methods on `character.Model`:
  - `Party() *party.Model` — returns nil (or sentinel) when the decorator hasn't been applied OR when the character is not in a party. Document the convention chosen.
  - `InParty() bool` — convenience boolean. Returns false when not decorated OR not in a party.
- In `kafka/consumer/character/consumer.go` `handleStatusEventStatChanged`, replace the inline party-announce provider chain with: fetch the character via `cp.GetById(cp.PartyDecorator)(...)`, short-circuit `if !c.InParty() { return }` before constructing any party announcement, and only then proceed with the existing `OtherMemberInMap` / `FilteredMemberProvider` / `ForEachByCharacterId` flow.
- The duplicate announcement path at `kafka/consumer/map/consumer.go:230` must be updated to the same shape.
- The `Unable to announce character [%d] health to party members.` debug log MUST be removed from both call sites. Any genuine errors from `ForEachByCharacterId` should be allowed to bubble up only if they represent something other than "no party members to announce to."

### 4.2 atlas-saga-orchestrator — Nil-UUID Short-Circuit

- At the saga consumer entrypoint(s), reject events whose `transactionId` is `uuid.Nil` (the zero UUID) before any saga lookup or DB read.
- The rejection MUST occur before any work that touches saga storage. The "Saga event skipped" log (with `reason: saga_not_found`) MUST NOT fire for nil-UUID events.
- Implementation may use a small helper (e.g., `IsCandidateTransactionId(uuid.UUID) bool`) or an in-line guard, at the implementer's discretion. The guard must be applied consistently across all consumer types that go through `event_acceptance.AcceptEvent` (or the equivalent entrypoint).
- For events with non-nil but unrecognized transaction IDs, current behavior is preserved (lookup → `saga_not_found` debug log).
- Emit a structured debug log at the new short-circuit point with reason `nil_transaction_id`, mirroring the existing `LogSkip` shape, so the suppression is observable. Reuse `LogSkip` if practical.

### 4.3 atlas-consumables — Parallel Independent Reads

- In `consumable.ConsumeStandard`, the three reads `cp.GetById(characterId)`, `character2.GetMap(characterId)`, and `cdp.GetById(itemId)` must be issued concurrently.
- The implementation must propagate errors faithfully: if any of the three fails, the resulting error must take the same `ConsumeError` path as today, with the same transaction cancellation behavior. Concurrent execution must not change the observable error semantics.
- The same pattern must be applied to other `Consume*` variants in the same file when and only when the initial reads in that variant are demonstrably independent. At minimum, evaluate `ConsumeTownScroll` and `ConsumeSummoningSack`. Where reads are not independent (e.g., one feeds into another's input), leave sequential and document the reason in a code comment.
- Choice between `model.ParallelMap` vs `errgroup`: prefer `model.ParallelMap` if a clean fit exists; otherwise `errgroup.WithContext` is acceptable. The choice must be consistent within `consumable.go` (do not mix styles for similar shapes).
- Trace propagation must continue to work: spans created during the parallel branches must remain children of the consume span. Verify post-implementation by re-running the trace investigation.

## 5. API Surface

No external HTTP API changes. No new Kafka topics. No JSON:API payload changes. No event payload changes.

Internal Go API additions:
- `atlas-channel/character.ProcessorImpl.PartyDecorator(m Model) Model` (new method, satisfies the existing decorator-method shape).
- `atlas-channel/character.Model.Party() *party.Model` and `Model.InParty() bool` (new accessors).
- Possibly `atlas-saga-orchestrator/saga.IsCandidateTransactionId(id uuid.UUID) bool` (helper, optional).

## 6. Data Model

No database schema changes. No migrations. No new persisted state.

The character model in atlas-channel gains an in-memory optional `party` field; this is a runtime decoration only.

## 7. Service Impact

| Service | Change |
|---|---|
| atlas-channel | New `PartyDecorator`, new `Party()` / `InParty()` getters on character model, updated `kafka/consumer/character` and `kafka/consumer/map` HP-announce paths, removed misleading log lines. |
| atlas-saga-orchestrator | Nil-UUID short-circuit at the consumer entrypoint(s), one new debug-log reason. |
| atlas-consumables | `ConsumeStandard` (and applicable siblings) refactored to issue independent reads concurrently. |
| atlas-character | No changes. |
| atlas-parties | No changes (its REST endpoint is consumed by the new decorator with no contract impact). |
| All other services | No changes. |

## 8. Non-Functional Requirements

### 8.1 Performance Target

- Measured on a fresh Tempo trace of a single solo HP-potion use (same flow as `665342ad8db44ec8ce4b5bb108e5de7e`), the **`CharacterItemUseHandle` root span (atlas-channel) p50 duration must be ≤ 200ms** when measured across at least 5 consecutive uses on a quiet test world.
- For comparison, the baseline trace shows ~368ms of visible activity and ~700ms of total root-span duration. The expected savings: ~50ms from parallelization in atlas-consumables, plus reduction of saga-orchestrator hop time on the critical path. The party-announce change does not affect this measurement directly (the announce work is on the EVENT_TOPIC_CHARACTER_STATUS consumer span, not the use-item root span), but it is in scope for the same task because it shares the same trace evidence.
- A secondary measurement: log volume for "Unable to announce character health to party members" and "Saga event skipped … reason=saga_not_found" with nil transaction ID over a 5-minute solo-play sample MUST drop to zero.

### 8.2 Observability

- Tracing MUST continue to function across the parallelized consumables branches. Trace context (`ctx`) MUST be passed into each branch; any goroutine started for parallel reads must inherit the parent context.
- Debug-level skip log for nil-UUID short-circuit MUST be present (so suppressed work is countable in Loki) but MUST use a distinct `reason` value (`nil_transaction_id`) from `saga_not_found`.
- No new Prometheus metrics in scope.

### 8.3 Multi-Tenancy

- All changes operate inside existing tenant-scoped contexts (`tenant.MustFromContext(ctx)`). No tenant-boundary crossings introduced.
- Concurrency added in atlas-consumables MUST NOT leak tenants between concurrent goroutines. Each goroutine MUST derive from the parent tenant-bearing context.

### 8.4 Testing

- atlas-channel: unit-test coverage for the new `PartyDecorator` showing both the in-party and not-in-party paths. Update existing `kafka/consumer/character` tests so the announce path is exercised under both conditions; the not-in-party path must not log the announce-failure line.
- atlas-saga-orchestrator: unit test for the nil-UUID short-circuit covering at least one consumer that uses `AcceptEvent` (or the equivalent entrypoint chosen). The test must assert no saga storage call is issued for a nil UUID.
- atlas-consumables: regression test for `ConsumeStandard` confirming all three reads are exercised, and that an error from any of them produces the existing `ConsumeError` outcome.
- All affected services MUST build cleanly (`go build ./...`) and pass existing tests.

## 9. Open Questions

- `Party()` return convention: pointer with nil-when-not-in-party, or non-pointer with an `IsZero()` check, or sentinel? Pick the one most consistent with existing decorator-loaded fields (`InventoryDecorator` etc.) — to be confirmed during design phase.
- Saga short-circuit placement: a single helper at the `AcceptEvent` boundary, or distributed guards in each consumer? Lean: helper at the boundary, but this is a design-phase call.
- Whether to backfill the same `model.ParallelMap`/`errgroup` pattern to `ConsumeScroll` is explicitly **deferred**: its reads are not independent (scroll uses character-with-inventory; equipment lookup depends on the result). Confirm during design.

## 10. Acceptance Criteria

A working implementation must satisfy all of the following, verifiable from a single fresh Tempo trace and the corresponding Loki logs:

1. ☐ A solo character (no party) uses an HP potion. Loki shows zero "Unable to announce character health to party members" log lines for that interaction.
2. ☐ The `kafka/consumer/character` and `kafka/consumer/map` HP-announce paths both go through `cp.GetById(cp.PartyDecorator)(...)` and short-circuit on `!c.InParty()`.
3. ☐ `character.Model` exposes both `Party()` and `InParty()`. Both have unit-test coverage for in-party / not-in-party / undecorated states.
4. ☐ Saga-orchestrator emits zero `saga_not_found` skip logs for events whose transaction ID is the zero UUID, in any 5-minute play sample. Such events instead surface the new `nil_transaction_id` reason at debug level (or are silent — the implementation's choice — but MUST NOT produce `saga_not_found`).
5. ☐ `consumable.ConsumeStandard` issues `cp.GetById`, `GetMap`, and `cdp.GetById` concurrently. The fresh trace shows the three child spans overlap rather than running sequentially.
6. ☐ Same parallelization applied to at least `ConsumeTownScroll` and `ConsumeSummoningSack` where reads are independent, or a code comment in each unmodified variant documenting why parallelization does not apply.
7. ☐ `CharacterItemUseHandle` root-span p50 across 5 consecutive solo HP-potion uses on a quiet test world is ≤ 200ms in Tempo.
8. ☐ All affected services build (`go build ./...`) and pass existing tests. New tests added per §8.4 pass.
9. ☐ No regression in the use-item flow: HP applied, item consumed, client receives buff/stat update, no orphaned transactions.
