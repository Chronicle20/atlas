# Backend Audit (Final) — task-119-db-transaction-coverage

- **Scope:** Full branch diff `560b4fcd0..20ca9f5d9` (24 commits). Changed non-test Go modules: `libs/atlas-database`; services `atlas-families`, `atlas-inventory`, `atlas-keys`, `atlas-maps`, `atlas-marriages`, `atlas-monster-book`, `atlas-npc-conversations`, `atlas-pets`, `atlas-storage`.
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-13
- **Build:** PASS — all 9 modules `go build ./...` clean.
- **Tests:** PASS — all 9 modules `go test ./... -count=1` clean (no FAILs).
- **go vet:** PASS on `atlas-inventory` and `atlas-pets` (empty output, zero findings).
- **Overall:** PASS — zero blocking findings.

## Build & Test Results

```
atlas-database:          go build clean; go test clean (per prior review-backend.md, re-confirmed transaction.go/transaction_test.go read this pass)
atlas-families:          go build clean; go test ok atlas-family/family 0.018s
atlas-inventory:         go vet clean; go build clean; go test ok (compartment, inventory, drop, kafka/message/compartment)
atlas-keys:               go build clean; go test ok atlas-keys/key 0.008s
atlas-maps:               go build clean; go test ok (20 packages incl. kafka/consumer/character)
atlas-marriages:          go build clean; go test ok (10 packages incl. marriage, scheduler)
atlas-monster-book:       go build clean; go test ok (card, collection, kafka/consumer/character, kafka/consumer/monsterbook)
atlas-npc-conversations:  go build clean; go test ok (conversation, conversation/npc, conversation/recipe, kafka/consumer/character)
atlas-pets:               go vet clean; go build clean; go test ok (character, kafka/consumer/character, kafka/consumer/pet, location, pet 2.37s)
atlas-storage:            go build clean; go test ok (asset, storage)
```

## Priority 1 — atlas-inventory tx-binding fixes

### `compartment/processor.go` (`p.WithTransaction(tx)` bindings)

| Site | Status | Evidence |
|------|--------|----------|
| `IncreaseCapacity` compartment read | PASS | `compartment/processor.go:630` — `p.WithTransaction(tx).GetByCharacterAndType(...)` inside the `ExecuteTransaction` closure that opened `tx`. |
| `Drop` compartment read | PASS | `compartment/processor.go:675` |
| `AttemptEquipmentPickUp` compartment read | PASS | `compartment/processor.go:1123` |
| `AttemptItemPickUp` nested `CreateAsset` calls (x2) | PASS | `compartment/processor.go:1271`, `:1287` — both rebind via `p.WithTransaction(tx).CreateAsset(...)` |
| `MergeAndCompact` compartment reads + `Move` calls | PASS | `compartment/processor.go:1422,1445,1448,1480,1483` |
| `CompactAndSort` compartment reads + `Move` calls | PASS | `compartment/processor.go:1755,1779,1782,1813,1816` |
| No double-wrap | PASS | `WithTransaction(tx)` (`compartment/processor.go:117-127`) rebinds `p.db = tx` on a shallow clone; the tx-bound clone's own `Move`/`CreateAsset` call `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)` (`compartment/processor.go:433`, `:1015`) where `p.db` is now `tx` — `isTransaction(tx.WithContext(...))` resolves true (same `*sql.Tx` `ConnPool`), so `ExecuteTransaction` runs `fn(db)` directly with no nested `BEGIN`. Verified against `libs/atlas-database/transaction.go:9-13`. |
| No lost sub-state | PASS | `WithTransaction` clone (`compartment/processor.go:117-127`) copies every field (`assetProcessor`, `dropProcessor`, `equipmentProcessor`, `producer`, `t`) — only `db` changes. |

### `inventory/processor.go` (rebind of compartment sub-processor)

| Check | Status | Evidence |
|-------|--------|----------|
| `WithTransaction` rebinds `compartmentProcessor` to the tx | PASS | `inventory/processor.go:52` — `compartmentProcessor: p.compartmentProcessor.WithTransaction(db)`. Prior to this diff it was `compartmentProcessor: p.compartmentProcessor` (unchanged reference), which meant a tx-bound `inventory.Processor` clone would dispatch compartment reads/writes through the ORIGINAL non-tx-bound sub-processor — a real bug this diff fixes. Confirmed via `git diff 560b4fcd0..20ca9f5d9 -- .../inventory/processor.go`. |

### Sweep for missed tx-binding spots

Walked every remaining un-`WithTransaction`-prefixed call to `GetByCharacterAndType`/`CreateAsset`/`Move` in `compartment/processor.go` (lines 424, 806, 1005, 1902): each resolves correctly — line 424/1005 are inside `MoveAndLock`/`CreateAssetAndLock`, which are only ever invoked via a `p.WithTransaction(tx)` receiver from their `*AndEmit` callers (`compartment/processor.go:411`, `:991`), so `p.Move`/`p.CreateAsset` there already dispatch through the tx-bound receiver; line 806 (`CancelReservation`) is a read-only path with no `ExecuteTransaction` wrapper at all; line 1902 uses the already-tx-bound local `cp` variable, not `p`. No missed sites found.

## Priority 1 — atlas-pets Despawner fix

| Check | Status | Evidence |
|-------|--------|----------|
| Construction-time binding removed | PASS | `pet/processor.go:113-117` (`NewProcessor`) no longer contains `p.Despawner = p.defaultDespawn`; diff confirms the line was deleted. |
| Dispatch falls through correctly on tx-bound clones | PASS | `pet/processor.go:558-563` — `Despawn()` checks `if p.Despawner != nil { return p.Despawner(mb) }` else `return p.defaultDespawn(mb)`; with `Despawner` left `nil` in production, every call — including on a `With(WithTransaction(tx))` clone — resolves `defaultDespawn` against the clone's own receiver (correct `db`). |
| Root-cause documented in code | PASS | `pet/processor.go:88-96` — comment explains `With()` shallow-copies (`clone := *p`, `pet/processor.go:187-189`) without rebinding method values, so a field bound at `NewProcessor` time keeps referencing the ORIGINAL receiver. |
| Regression test exercises the real path (not the mock) | PASS | `pet/processor_test.go:1610-1646` (`TestProcessor_DespawnAndEmit_RidesOuterTransaction`) forces the outer transaction's second write (`outbox_entries`) to fail via `FailWritesOn` and asserts the despawn's slot write rolled back too — this fails under the pre-fix code (despawn escapes into its own separate transaction and survives the outer rollback) and passes post-fix. |
| Other method-value fields swept for the same latent bug | PASS | Searched all 9 changed services for `= p.<MethodName>` (no-call, bare method-value assignment) patterns and `NewProcessor`-time bindings mirroring the `Despawner` shape: zero other hits. `pet/processor.go:116` `p.rollEvolution = weightedRoll` is safe — `weightedRoll` (`pet/processor.go:124`) is a free function taking `[]uint32`, not a method value on `p`, so it carries no stale receiver. |

## Priority 2 — libs/atlas-database core fix

Already reviewed in this task's own prior pass (`docs/tasks/task-119-db-transaction-coverage/review-backend.md`, dated 2026-07-12, PASS, no blocking findings). Re-read `libs/atlas-database/transaction.go` this pass: `isTransaction` (lines 15-23) matches GORM's own `TxCommitter` type-assertion idiom; `ExecuteTransaction` (lines 6-12) correctly branches join-vs-open. No new findings.

## Priority 3 — remaining 7-service remediation (spot-check)

All mechanical `db.Transaction(...)` → `database.ExecuteTransaction(...)` swaps inspected are behavior-preserving (gain join-semantics, no other change):

- `atlas-families/family/processor.go:176,248,321` (`AddJunior`, `RemoveMember`, `BreakLink`)
- `atlas-keys/key/processor.go:75,93,107,119` (`Reset`, `CreateDefault`, `Delete`, `ChangeKey`)
- `atlas-npc-conversations/conversation/npc/processor.go:133,156,180,202,222,278` (create/update/delete/reindex paths)
- `atlas-monster-book/kafka/consumer/character/consumer.go:50` (`handleStatusEventDeleted`)
- `atlas-maps/kafka/consumer/character/consumer.go:189` (`handleStatusEventDeletedFunc`) — additionally fixed to make the visit-delete + location-delete pair atomic (previously two independent non-transactional calls, each logging-and-continuing on error; now one `ExecuteTransaction` with early-return on error).
- `atlas-marriages/marriage/processor.go:243-317` (`AcceptProposal` wrapped in `ExecuteTransaction`; dead `AcceptProposalWithTransactionAndEmit` + `executeInTransaction` helper removed — confirmed zero remaining references via `grep -rn executeInTransaction services/atlas-marriages`).
- `atlas-storage/storage/processor.go`: `WithTransaction` added (`:68-71`, correctly placed in `processor.go`, matches FILE-01); `MergeAndSort` restructured to prefetch `atlas-data` slot-max lookups (network I/O) before opening the transaction (`:551-560`, commented rationale) then wraps all writes in one `ExecuteTransaction` (`:562-608`); `ExpireAndEmit` wraps delete+replacement-create in one transaction and defers `emitExpiredEvent` until after commit (`:764-806`, comment: "Publish only after the transaction commits: no event for a rolled-back expiry"); `DeleteByAccountId` wraps the per-storage delete loop in one transaction with early-return on error (previously logged-and-continued per storage, silently leaving partial deletes).

## DOM-24 (Kafka producer stubbing) — checked against the two `*_rollback_test.go` files that call `*AndEmit`

`atlas-marriages/marriage/processor_rollback_test.go:34` and `atlas-storage/storage/processor_rollback_test.go:32` call `AcceptProposalAndEmit`/`ExpireAndEmit` under an injected write failure. Both call sites read the emit-triggering code path: `AcceptProposalAndEmit` (`marriage/processor.go:325-328`) and `ExpireAndEmit` (`storage/processor.go:794-798`) both `return` immediately when the wrapped `ExecuteTransaction`/inner call errors, *before* reaching `message.Emit`/`p.emitExpiredEvent` — so the fault-injected failure path never reaches a real producer call. Confirmed by running `TestProcessor_DespawnAndEmit_RidesOuterTransaction` (the equivalent pets case) at 0.00s wall time — consistent with the outbox pattern (task-114, merged before this branch) routing emits through an in-transaction `outbox_entries` DB write rather than a direct Kafka call, so no producer-stub gap exists on this branch. No finding.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None identified beyond what the branch's own prior `audit.md`/`review-backend.md`/`review-plan-adherence.md` already recorded (Task 1-3 code reviews, D0 blast-radius closeout). This pass corroborates those findings independently for the two highest-risk, not-previously-DOM-reviewed fixes (atlas-inventory tx-binding, atlas-pets Despawner) and finds them correct.
