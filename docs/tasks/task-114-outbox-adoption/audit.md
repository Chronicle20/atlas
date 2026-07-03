# Plan Audit — task-114-outbox-adoption

**Plan Path:** docs/tasks/task-114-outbox-adoption/plan.md
**Audit Date:** 2026-07-03
**Branch:** task-114-outbox-adoption
**Base Branch:** main
**Range audited:** 38fcddc..a46d3df2f (38 commits)

## Executive Summary

All 26 planned tasks were implemented and are individually traceable to commits;
one out-of-scope service (atlas-monster-book) was additionally migrated after
`tools/outboxguard` flagged it. The lib changes (header round-trip, id-ordered
publish, `EnqueueBuffer`, `EmitProvider`, `TopicWriterPool` promotion) are present
and their tests pass; the per-service migrations use the correct
`database.ExecuteTransaction` + `outbox.EmitProvider(tx)` seam so the atomicity
guarantee will hold once task-119 lands. The single substantive divergence from
the fleet norm is **atlas-quest, which swallows outbox-enqueue errors on its 6
status-event emit sites** (log + `return nil`) rather than propagating them to
roll back the transaction — this silently defeats task-114's atomicity intent for
those events and should be fixed. It is not strictly PR-blocking because the whole
guarantee is latent until task-119, but it is an Important consistency defect.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Header round-trip + id-ordered publish | DONE | `libs/atlas-outbox/headers.go` (base64 encode/decode); `drainer.go:220,224` `Order("id ASC")`; `drainer.go:234-243` re-attaches headers |
| 2 | Promote TopicWriterPool; swap configurations | DONE | `libs/atlas-outbox/publisher.go` present; commit 866c544e2; configurations `main.go` uses `outboxlib.NewTopicWriterPool()` |
| 3 | Lib deps + EnqueueBuffer bridge | DONE | `libs/atlas-outbox/bridge.go` (`EnqueueBuffer`, `headerMap`); `bridge_test.go` passes |
| 4 | EmitProvider | DONE | `libs/atlas-outbox/provider.go`; `provider_test.go` passes |
| 5 | Lib README + verification | DONE | commit f3cddc5a5; `go test ./...` = ok |
| 6 | Wire outbox into atlas-character | DONE | commit e286be5a0; migration + drainer in main.go |
| 7 | Three meso paths | DONE | `character/processor.go:782` `RequestChangeMeso` matches plan (rejectEmit closure, `ErrMesoOverflow`, in-tx `outbox.EmitProvider`); `meso_outbox_test.go` |
| 8 | Fame + AP distribution | DONE | commit a69433152; `RequestChangeFame`/`RequestDistributeAp` migrated, rejection emits post-tx |
| 9 | Invert every remaining Emit site | DONE | commit 6bf4a2218; 25 `*AndEmit` sites migrated per inventory |
| 10 | Acceptance tests + inventory start | DONE | `outbox_acceptance_test.go` (rollback test `t.Skip`'d pending task-119, documented); inventory.md created |
| 11 | atlas-inventory | DONE | commit 0e5fedc87 + 3 D7 fix commits; 22 migrated / 7 left-direct |
| 12 | atlas-cashshop | DONE | commits 83f312b22 + 83b549634 (hand-rolled flush-loop fix pass) |
| 13 | atlas-fame | DONE | commit 2d30c4838; `WithTransaction` added, rejectEmit device |
| 14 | atlas-buddies | DONE | commit 1ac427853 |
| 15 | atlas-guilds | DONE | commit fd37823ba |
| 16 | atlas-notes | DONE | commits fbe58604a + 8d6d99c96 (saga commands fire post-commit) |
| 17 | atlas-pets | DONE | commit b4465b290 |
| 18 | atlas-skills | DONE | commit b0468566a |
| 19 | atlas-merchant | DONE | commits 0d58bb6df + 380531fe7 (Frederick error propagation) |
| 20 | atlas-npc-shops | DONE | commit 02f3600a5 |
| 21 | atlas-tenants | DONE | commit 82e6927ca |
| 22 | atlas-mounts | DONE | commit 9a69fe10f (Pattern C) |
| 23 | atlas-quest (EventEmitter) | DONE (with caveat) | commit 14e2b8509; `outbox_event_emitter.go` + `txEmitter` plumbing. See Finding 1 — status emits swallow enqueue errors |
| 24 | gachapons/drop-information/data inventory-only | DONE | commit 50c2f5f93; inventory sections present |
| 25 | outboxguard analyzer + wrapper + CI | DONE | commits d1752251a + 3a1fbdbe2 (nested-funclit skip); tests pass |
| 26 | Fleet verification + CD-2 closeout | DONE | commit a46d3df2f; CD-2 = RESOLVED (task-114); inventory final sweep header |

**Completion Rate:** 26/26 tasks (100%), plus 1 bonus service (atlas-monster-book).
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 23 complete but diverges — see Findings)

## Skipped / Deferred Tasks

None. Every plan task has a corresponding commit and verified artifact. Two items
are deferred *by design* and documented:

- **Atomicity is latent until task-119** (`ExecuteTransaction` no-op). Not a task-114
  gap — the migrations use the correct seam and become atomic with zero code change
  once task-119 lands. Accurately documented in CD-2 and inventory.md.
- **Consumer dedup on TransactionId** is explicitly out of scope, tracked as CD-1.

## Cross-Cutting Assessments

### 1. task-119 dependency (documented, accurate) — CONFIRMED

The CD-2 closeout (`docs/architectural-improvements.md:218`) and inventory.md both
state clearly that `database.ExecuteTransaction` is a verified no-op today
(`isTransaction` true because `gorm.Open` populates `Statement.ConnPool`), so no
real BEGIN/COMMIT/ROLLBACK wraps enqueue+write until task-119's TxCommitter fix
merges. Spot-checked that migrations use the correct seam: `character/processor.go:782`
issues both the `dynamicUpdate` write and `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))`
inside the same `ExecuteTransaction(tx)` closure; monster-book `consumer.go:57` and
`card/processor.go:102` do the same. The guarantee will hold once task-119 lands.
The `TestOutbox_RollbackDiscardsEnqueuedEvents` skip is honestly explained.

### 2. atlas-quest swallows outbox-enqueue errors — CONFIRMED DIVERGENCE (Important)

All 6 quest **status**-event emit sites (`quest/processor.go:229, 376, 480, 552, 601,
609`) are structured as:

```go
if err := p.txEmitter(tx).EmitQuestStarted(...); err != nil {
    p.l.WithError(err).Warnf("Unable to emit ...")
}
return nil
```

The enqueue error is logged and dropped; the closure returns nil, so the transaction
commits regardless. This diverges from the fleet norm where `message.Emit` returns
the enqueue error out of the closure → tx rolls back. It is also **internally
inconsistent**: the 2 saga-command sites (`:895, :965`) do
`return awardedItems, p.txEmitter(tx).EmitSaga(s)`, which *does* propagate the error.

**Impact:** once task-119 lands, a quest status enqueue failure will still commit the
domain write while losing the event — precisely the CD-2 "commit happened but event
lost" failure mode task-114 exists to prevent. Because `EnqueueBuffer` failure inside
a real transaction should abort it, best-effort here is not defensible for a
state-asserting event.

**Recommendation:** bring quest in line — change the 6 status sites from log-and-continue
to `return err` (mirroring the saga sites and the fleet `message.Emit` contract). Low
effort, no interface change. Not strictly PR-blocking (guarantee latent until task-119),
but should land before or with task-119 so quest isn't silently exempt.

### 3. Command-to-other-service emits routed through the outbox — CONSISTENT / ACCEPTABLE

The D7 tension is handled consistently across services: a cross-service command that is
*causally coupled to a committed write* rides the same outbox buffer and drains
post-commit (inventory `RequestPickUp` success path; quest saga commands; character
`AwardExperience`'s follow-on `awardLevelCommand`), while a command that fires *because
a write did NOT happen* (rollback/rejection) is routed direct on a throwaway buffer
(inventory `CancelReservation`, `Accept`/`Release` failure branches; character/fame
rejectEmit closures). The atlas-inventory D7 fix-pass (commits 0e3aa3a52, b820a3db7,
83aa14a28) explicitly split these and updated tests to assert the failure-path command's
*absence* from the outbox buffer. This is a coherent, defensible fleet-wide rule.

### 4. atlas-monster-book (out of original scope) — CONFIRMED COMPLETE & CORRECT

Migration is complete: all 3 in-tx direct emits migrated to `outbox.EmitProvider`
(`card/processor.go:102`, `collection/processor.go:257`,
`kafka/consumer/monsterbook/consumer.go:58`), drainer + migration wired in `main.go:56,61`,
`WithTransaction` sub-processor rebinding verified. Notably the migration also fixed a
**silently-masked broken test** (`TestHandleCardPickedUpInsertsAndRecomputes` was passing
despite writing to a non-existent `outbox_entries` table, because the no-op tx left prior
writes committed and the handler only logged the error) by adding `outbox.Migration(db)`
and a real outbox-row-count assertion. Good catch, well documented.

### 5. Dead code left behind — CLEANUP CANDIDATES (non-blocking)

- **quest** `KafkaEventEmitter` / `NewKafkaEventEmitter` (`quest/event_emitter.go:25-33`)
  and the `eventEmitter` field set at `processor.go:100`: in the default `NewProcessor`
  path the `txEmitter` wraps `NewOutboxEventEmitter`, so the production `KafkaEventEmitter`
  is never used for emission. The `eventEmitter` field is still live only for the
  mock-injection path (`NewProcessorWithDependencies:118`). Candidate for removal once
  the mock path is refactored to inject a `txEmitter` directly.
- Dead struct-field `producer.Provider` initializers documented in inventory: cashshop
  (`cashshop/inventory/processor.go:47`, `.../compartment/processor.go:60`,
  `cashshop/processor.go:72`, `wallet/processor.go:50`, `wishlist/processor.go:44`),
  npc-shops (`kp`), mounts (`kp`). All confirmed unread by any emit path (documented in
  inventory Task 26 sweep). Benign; remove in a follow-up.

## Build & Test Results

Spot-checked (Task 26 reported full-fleet green: `docker buildx bake all-go-services`,
`tools/outbox-guard.sh`, `tools/redis-key-guard.sh`, per-module `-race`/`vet`/`build`):

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| libs/atlas-outbox | PASS | PASS | `go test ./...` = ok |
| tools/outboxguard | PASS | PASS | `GOWORK=off go test ./...` = ok (analyzer + nested-funclit skip) |
| atlas-character | PASS | (not re-run) | `go build ./...` clean; reference impl verified by inspection |
| atlas-monster-book | (reported) | (reported) | migration verified by inspection; inventory records green gates |

I did not re-run the full 18-module `-race` matrix or the docker bake (expensive; reported
green in Task 26 with the correct commands). The lib + guard spot-checks and the
character build hold up.

## Overall Assessment

- **Plan Adherence:** FULL — 26/26 tasks implemented and evidenced; scope expansion
  (monster-book) handled correctly rather than deferred.
- **Recommendation:** NEEDS_REVIEW — mergeable, but the atlas-quest enqueue-error
  swallowing (Finding 1) should be resolved so quest is not silently exempt from the
  atomicity contract when task-119 lands.

## Action Items

1. **(Important)** atlas-quest: propagate outbox-enqueue errors on the 6 status-event
   sites (`quest/processor.go:229, 376, 480, 552, 601, 609`) — replace log-and-`return nil`
   with `return err`, matching the saga sites (`:895, :965`) and the fleet `message.Emit`
   contract. Update any test that asserts best-effort behavior.
2. **(Cleanup, non-blocking)** Remove quest's dead `KafkaEventEmitter`/`eventEmitter`
   field once the mock path injects `txEmitter` directly.
3. **(Cleanup, non-blocking)** Remove the dead `producer.Provider` struct fields in
   cashshop (×5), npc-shops (`kp`), mounts (`kp`).
4. **(Doc hygiene, non-blocking)** The plan.md checkboxes were never marked `[x]`
   (0 of the `- [ ]` boxes flipped) even though all tasks completed — update or note,
   since a reader scanning plan.md would wrongly conclude nothing was done.
5. **(Guard-precision note)** outboxguard skips nested func literals (commit 3a1fbdbe2)
   to avoid false-positiving deferred rejectEmit closures; this leaves a blind spot for a
   nested func literal *synchronously invoked* inside a tx. Acceptable given the fleet's
   uniform patterns, but worth a comment in the analyzer if not already covered.
