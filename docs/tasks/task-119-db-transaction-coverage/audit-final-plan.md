# Plan Audit (Final) — task-119-db-transaction-coverage

**Plan Path:** docs/tasks/task-119-db-transaction-coverage/plan.md
**Audit Date:** 2026-07-13
**Branch:** task-119-db-transaction-coverage
**Merge base:** 560b4fcd0 (24 commits ahead)

## Task-by-Task Verdicts

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | `isTransaction` uses `gorm.TxCommitter` + regression tests | DONE | `libs/atlas-database/transaction.go:17-27` (`committer, ok := db.Statement.ConnPool.(gorm.TxCommitter)`); `libs/atlas-database/transaction_test.go` present, 4 tests. Commit `795afce28`. |
| 2 | `databasetest.FailWritesOn` helper + tests | DONE | `libs/atlas-database/databasetest/failwrites.go` (`WriteCreate/Update/Delete` verbs, callback registration), `failwrites_test.go` present. Commit `3f8bce38f`. |
| 3 | Full 14-service write-path audit → audit.md | DONE | `audit.md` (536 lines), one section per all 14 services (npc-conversations, keys, families, marriages, monster-book, storage, account, ban, maps, map-actions, portal-actions, reactor-actions, party-quests, saga-orchestrator) with write-inventory/exclusions/verdicts tables. Commit `b28e0d1b2`. |
| 4 | Rebase gate recorded (task-114/116 merged) | DONE | `audit.md:460-475` "Rebase gate (Task 4) — 2026-07-12": rebased onto `e15b343b1`, confirms task-114 (`d2e13ba3d`) and task-116 (`e15b343b1`) merge commits, flags that main's pre-fix `isTransaction` made task-114's outbox enqueue-in-tx non-atomic until this branch lands, and documents the emit-convention re-check (monster-book migrated to outbox by task-114, others not). Commit `8bf3d1d52`. |
| 5 | atlas-keys: 4 raw tx → ExecuteTransaction + rollback test | DONE | `grep '\.Transaction('` on `key/processor.go` returns 0 hits (all 4 sites converted); `key/processor_rollback_test.go` exists. Commit `e01c5639c`. |
| 6 | atlas-families: 3 raw tx → ExecuteTransaction + rollback test | DONE | Same grep 0 hits on `family/processor.go`; `family/processor_rollback_test.go` exists. Commit `a1812681d`. |
| 7 | atlas-npc-conversations: 6 raw tx → ExecuteTransaction + rollback test | DONE | Same grep 0 hits on `conversation/npc/processor.go`; `processor_rollback_test.go` exists. Commit `47767eb6c`. |
| 8 | atlas-monster-book: emit/tx inversion, 2 consumers + rollback test | DONE (adapted) | `kafka/consumer/character/consumer.go:50` and `kafka/consumer/monsterbook/consumer.go:57` both call `database.ExecuteTransaction`; `consumer_rollback_test.go` present. Deviation from plan text is explicitly justified in audit.md's rebase gate: task-114 had already migrated monster-book to the outbox, so Task 8 used the outbox `EmitProvider` enqueue-in-tx shape instead of buffer+publish-after-commit — a documented, correct adaptation per the plan's own Task 4 Step 4 contingency. Commit `8cc9e0536`. |
| 9 | atlas-marriages: eliminate manual Begin/Commit, live flow wrapped, dead twin deleted | DONE | `marriage/processor.go` has zero `.Begin()/.Commit()/.Rollback()` hits; `database.ExecuteTransaction` present (line 245); only `AcceptProposal`/`AcceptProposalAndEmit` remain (the old `AcceptProposalWithTransactionAndEmit` + `executeInTransaction` dead twin is gone — function no longer present). `processor_rollback_test.go` exists. Commit `9905815a1`. |
| 10 | atlas-storage: `WithTransaction` plumbing + `GetOrCreateStorageId` wrap | DONE | `storage/processor.go:69` and `asset/processor.go:38` both define `WithTransaction`; `asset/processor.go:68` wraps `GetOrCreateStorageId` in `database.ExecuteTransaction`; `asset/processor_rollback_test.go` exists. Commit `c7a298f04`. |
| 11 | atlas-storage: `ExpireAndEmit` wrapped, emit after commit | DONE | `storage/processor.go:764` wraps `ExpireAndEmit` body in `database.ExecuteTransaction`; `storage/processor_rollback_test.go` exists. Commit `9ad4e5d16`. |
| 12 | atlas-storage: `MergeAndSort` wrapped | DONE | `storage/processor.go:562` wraps `MergeAndSort` in `database.ExecuteTransaction`. Commit `a607738a2`. |
| 12b | atlas-storage: `DeleteByAccountId` wrapped | DONE | `storage/processor.go:839` wraps `DeleteByAccountId` in `database.ExecuteTransaction`. Commit `7fd31b804`. |
| 13 | atlas-maps dual-delete wrapped; account/ban verdicts finalized | DONE | `kafka/consumer/character/consumer.go` character-deletion cleanup now single `database.ExecuteTransaction` (per commit diff, character_map_visits + character_locations deletes unified) with `consumer_rollback_test.go` added; `audit.md` atlas-account section documents `GetOrCreate` name-race as C+annotation (schema fix out of scope, correctly justified not silently dropped); atlas-ban section formally refutes the ban↔history pairing hypothesis with full call-chain trace. Commit `8c792ada7`. |
| 14 | audit.md closeout + architectural-improvements.md annotation | DONE | Commit `10f7d1b38` "close out audit — pointer convention, D0 blast-radius preamble, DL-4/CD-2 annotation"; `docs/architectural-improvements.md` shows as modified in branch. |
| 15 | Fleet verification — D0 blast-radius + fleet-wide caller audit | DONE | Commit `20ca9f5d9` adds 60-line "D0 blast-radius fleet audit" section to audit.md; concrete bugs it surfaced were fixed in commits `e5c6f38e8` (inventory: nested tx-opening calls bound to enclosing tx — root-handle escape bug), `256616e5a` (pets: `Despawn` method-value capture bug caused writes to escape caller's tx), `838ddbbbb` (inventory: compartment read bound to tx), and `36b23579f` (quest/character test harness moved to shared-cache in-memory sqlite because real transactions broke the old per-connection `:memory:` DB assumption) — this is exactly the "test the 18 activated-semantics modules, fix real bugs the D0 fix exposes" instruction in plan Task 15 Step 2, not skipped. |
| 16 | Code review + PR | PARTIAL/IN-PROGRESS | `review-backend.md`, `review-backend.json`, `review-plan-adherence.md` exist in the task folder (code review artifacts from an earlier pass, commit `e6af85291` "code-review artifacts for Tasks 1-3"). No evidence a full-branch review or PR was run post the Task 15 D0 fixes (commits after `10f7d1b38`). This audit itself is discharging part of Task 16's plan-adherence review; a backend-guidelines pass on the newest commits (inventory/pets fixes) and the actual PR opening still need to happen. |

## Build/Test Verification (sample modules, run this session)

| Module | Build | Test (`-race`) |
|---|---|---|
| `libs/atlas-database` | PASS | PASS — `ok github.com/Chronicle20/atlas/libs/atlas-database 1.064s`, `ok .../databasetest 1.032s` |
| `services/atlas-inventory/atlas.com/inventory` | PASS | PASS — all packages with tests (`asset`, `compartment`, `drop`, `inventory`, `kafka/message/compartment`) green |
| `services/atlas-pets/atlas.com/pets` | PASS | PASS — all packages with tests (`character`, `kafka/consumer/character`, `kafka/consumer/pet`, `location`, `pet`) green |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE. Tasks 1-15 are all faithfully implemented with strong file:line/commit evidence, including a legitimate, well-documented deviation on Task 8 (outbox adaptation) and real bugs caught and fixed during Task 15's fleet verification (inventory/pets tx-escape bugs) — exactly the kind of finding that task was designed to surface, not a sign of incompleteness.
- **Task 16 gap:** the code-review artifacts in the task folder predate the Task 15 fleet-fix commits (`e5c6f38e8`, `256616e5a`, `838ddbbbb`, `36b23579f`, `20ca9f5d9`). A fresh `superpowers:requesting-code-review` pass covering the full branch diff, plus opening the PR, has not yet run.
- **No skips, stubs, or TODOs found** in any of the 24 branch commits reviewed.

## Action Items

1. Run `superpowers:requesting-code-review` (plan-adherence-reviewer + backend-guidelines-reviewer) over the full branch diff, specifically covering the Task 15 fleet-fix commits, before opening the PR (Task 16 Step 1).
2. Complete Task 16 Step 2 (`superpowers:finishing-a-development-branch`) and open the PR once review is clean.
3. Consider whether the plan's own §2.4 recommendation (cutting the Task 1 lib fix into a standalone PR ahead of the full task-119 PR) is still desired — audit.md's rebase gate notes main is currently running the no-op `ExecuteTransaction` in production with task-114's outbox already merged on top of it, which is a live correctness gap until this branch (or at least commit `795afce28`) merges.
