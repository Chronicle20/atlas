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

---

# Backend Guidelines Audit (DOM-*/SUB-*/SEC-*) — 2026-07-03

Adversarial backend-guidelines pass over the task-114 migration surface (new
`libs/atlas-outbox`, `tools/outboxguard`, 16 service processor/consumer
changes). Builds on the plan-adherence section above; does not re-litigate the
accepted context (task-119 atomicity latency; D7 command-routing). Scope is the
migration delta, not a ground-up re-audit of untouched domain files.

## Verdict

**NEEDS-WORK (non-blocking cleanups only).** No Critical or Important DOM-*/SEC-*
finding blocks the PR. The uniform migration pattern is applied correctly: the
Processor Interface+Impl shape is preserved, `WithTransaction(tx)` rebinding is
correct (including sub-processor rebinds), tenant/span headers ride the outbox
path exactly as the direct path derives them, and enqueue errors propagate to
fail the enclosing transaction on the state-asserting emit sites. Immutable
models / Builder pattern are untouched. New lib + tool `go build`/`go vet`
clean. Remaining items are cleanup (dead fields, one stricter-validation
divergence, one guard blind spot) plus two corrections to the prior audit.

## DOM checklist assessment (migration-scoped)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Processor Interface+Impl preserved | new methods keep the pattern | PASS | quest `processor.go:36-134`; `WithTransaction` returns `Processor`, copies all fields incl. sub-processors |
| `WithTransaction(tx)` rebinds sub-processors | tx-scoped sub-processors, no stale `db` | PASS | cashshop `cashshop/inventory/compartment/processor.go:72` (`astP: asset.NewProcessor(p.l,p.ctx,tx)`); character `character/processor.go:161-163` (`sdp` copy fix landed) |
| Tenant context flows through outbox | headers carry tenant; table not tenant-scoped by design | PASS | `libs/atlas-outbox/bridge.go:41-57` merges `SpanHeaderDecorator`+`TenantHeaderDecorator(ctx)`; `entity.go` has no `tenant_id` (intentional — tenancy in headers, byte-exact via base64 `headers.go:11-30`) |
| Enqueue errors propagate → tx rollback | state-asserting emits return err | PASS | `libs/atlas-outbox/provider.go:22-28` returns err; quest status sites `processor.go:229,377,482,555,605,614` now `return err` (see Update A) |
| DOM-21 no reinvented shared types/consts | lib uses shared types | PASS | outbox lib imports `atlas-constants/world`, `atlas-kafka`, `kafka-go`; only new const is `notifyChannel="atlas_outbox_new"` (`outbox.go:50`) — no item/inventory/id types redeclared |
| Dead code after refactoring | anti-patterns.md line 35 | FAIL (minor) | see M3 |
| DOM-24 Kafka producer stubbed in emitting tests | shared `producertest` | PASS | `producertest.InstallNoop()` present in migrated emitting test pkgs (character, inventory, monster-book, buddies `testmain_test.go`/`processor_test.go`) |
| SEC-04 no hardcoded secrets in new lib/tool | grep | PASS | lib reads `BOOTSTRAP_SERVERS` (`publisher.go:26`) + injected DSN (`drainer.go:53`); no embedded keys/passwords. SEC-01..03 N/A (no auth/token/redirect surface) |

## Updates / corrections to the prior (plan-adherence) findings

**Update A — Prior Finding #2 (atlas-quest swallows status enqueue errors) is
RESOLVED in the current tree.** All six quest status-event sites now
log-and-`return err` inside the tx closure, matching the fleet `message.Emit`
contract: `quest/processor.go:231, 379, 484, 557, 607, 616`. The prior audit was
written against an earlier commit; the "log-and-return nil" divergence no longer
exists. Action item #1 can be closed.

**Correction B — Prior Finding #5 / Action item #3 wrongly lists atlas-npc-shops
`kp` as dead.** `kp` (`shops/processor.go:79`, init `:92`) is READ at five
`message.Emit(p.kp)` sites: `:268` (`EnterAndEmit`), `:287` (`ExitAndEmit`),
`:369` (`BuyAndEmit`), `:480`, `:570`. All are left-direct command relays with
no local DB write (the DB-writing methods `UpdateShop:194`/`DeleteAllShops:312`
emit nothing; `Enter`/`Exit`/`Buy` forward commands to other services), so `kp`
is legitimately live and correctly left direct. Remove npc-shops from the
dead-field cleanup list. No missed migration in npc-shops (no `p.kp` emit sits
inside a `database.ExecuteTransaction` with a write).

## New minor findings (non-blocking)

**M3 — Dead `producer.Provider` struct fields (anti-pattern: "Leaving dead code
after refactoring").** Confirmed unread by any emit path, yet still constructed
via `producer.ProviderImpl(l)(ctx)` at `NewProcessor` and (where present)
copied through `WithTransaction`:
- atlas-cashshop ×5: `wallet/processor.go:41` (init `:50`, copied `:61`),
  `wishlist/processor.go:35` (init `:44`, never copied/read — fully dead),
  `cashshop/processor.go:54` (init `:72`, never copied/read — fully dead),
  `cashshop/inventory/processor.go:36` (init `:47`, copied `:60`),
  `cashshop/inventory/compartment/processor.go:50` (init `:60`, copied `:71`).
- atlas-mounts ×1: `mount/processor.go:36` `kp` (init `:45`, never read).
- atlas-quest: `KafkaEventEmitter`/`NewKafkaEventEmitter`
  (`quest/event_emitter.go:25-33`) constructed at `processor.go:100` and copied
  at `:131`, but never invoked in production (all emits go through
  `txEmitter`→`OutboxEventEmitter`). Live only for the mock-injection seam
  (`NewProcessorWithDependencies:117-119`). Remove once the mock path injects a
  `txEmitter` directly.
- NOTE: cashshop `cashshop/inventory/asset/processor.go:44` `p` is NOT dead — it
  is read at `:202`/`:218` for the two left-direct no-op emits. Leave it.

**M1 — `outbox.Enqueue` validation is stricter than the direct producer (latent
behavior divergence).** `Enqueue` returns an error for an empty topic or an
**empty message key** (`libs/atlas-outbox/outbox.go:20-25`), which propagates
out of the emit closure and (post-task-119) rolls back the domain write. The
direct producer path performs no such check — `libs/atlas-kafka/producer/
producer.go` `Produce`/`tryMessage` writes whatever key the message carries,
including nil/empty. Any migrated event whose provider builds a `kafka.Message`
with no `Key` will now fail its transaction where it previously published. No
concrete empty-key emitter was found among the migrated services (the fleet norm
is `producer.CreateKey(...)`, always 8 bytes), so this is a latent robustness
risk, not a confirmed regression. Recommend a one-time grep of the migrated
`producer.go` files for `kafka.Message{` constructions omitting `Key` before
task-119 makes the rollback real.

**M2 — atlas-quest saga-command enqueue error is swallowed at the caller
(narrow atomicity gap for quest rewards).** `processStartActions`/
`processEndActions` correctly return the `EmitSaga` enqueue error
(`quest/processor.go:901, 971`), but both callers discard it and continue:
`processor.go:207-210` (Start) and `:468-471` (Complete) log a Warn and fall
through with "Don't fail the quest start/completion." Because the status-event
emit that follows in the same tx also writes `outbox_entries`, a *DB-level*
failure would still be caught there and roll back; the residual gap is a
saga-*specific* enqueue failure (e.g. saga command-topic resolution error via
`topic.EnvProvider`) that would let the quest commit with the reward saga
lost — precisely the CD-2 failure mode for the item/exp/meso/fame/skill grants.
Pre-existing best-effort policy on start/end actions, unchanged in mechanism by
the migration; low real-world likelihood. Optional: propagate the saga error for
reward-bearing completions, or document the carve-out.

**M4 — `outboxguard` blind spot + overstated doc claim.** The analyzer detects
only the literal `producer.ProviderImpl` selector inside a tx closure
(`tools/outboxguard/analyzer.go:43-56`) and deliberately does not descend into
nested func literals (`:36-42`). It therefore cannot catch a direct emit made
through a *stored* `producer.Provider` field (`message.Emit(p.p)(...)`) or a
hand-rolled `for t,ms := range mb.GetAll(){ p.p(t)(...) }` loop inside a tx — the
exact shapes the cashshop fix-pass had to find by manual grep. The analyzer's doc
comment (`:19-22`) asserts "the fleet's only direct-producer entry point in
service code is the local kafka/producer.ProviderImpl," which the cashshop
hand-rolled loops disprove. Fine as a `ProviderImpl`-regression tripwire, but the
comment overstates coverage; a future stored-provider in-tx emit would pass the
guard silently.

## Build / vet (new components)

- `libs/atlas-outbox`: `go build ./...` + `go vet ./...` clean.
- `tools/outboxguard`: `GOWORK=off go build ./...` clean.
- Did not re-run the 18-module `-race` matrix or docker bake (reported green in
  inventory Task 26; plan-adherence section spot-checked).

## Action items (supersedes/augments prior list)

1. ~~(Important) quest status-emit error propagation~~ — **DONE** (Update A;
   verified `processor.go:231,379,484,557,607,616`).
2. (Cleanup) Remove dead fields per M3: cashshop ×5, mounts `kp`, quest
   `KafkaEventEmitter`. **Do NOT** remove npc-shops `kp` or cashshop asset `p`
   (Correction B / M3 note — both live).
3. (Robustness) M1 — confirm no migrated event emits with an empty key before
   task-119, or relax the `Enqueue` empty-key guard to match the direct path.
4. (Optional) M2 — decide whether reward-bearing quest saga enqueue failures
   should roll back the quest, or document the best-effort carve-out.
5. (Doc) M4 — soften the `outboxguard` doc comment; note the stored-provider
   blind spot.
