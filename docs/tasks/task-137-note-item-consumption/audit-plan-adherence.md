# Plan Adherence Audit — task-137-note-item-consumption

**Plan Path:** docs/tasks/task-137-note-item-consumption/plan.md
**Audit Date:** 2026-07-25
**Branch:** task-137-note-item-consumption
**Merge base:** b64ee7936 (HEAD 619bb66ea, 25 commits)
**Auditor:** plan-adherence-reviewer (read-only)

## Executive Summary

All 18 plan tasks are IMPLEMENTED with file:line/commit evidence, corroborated by the
per-task `report.md`/`brief.md` pairs in `.superpowers/sdd/` and independently spot-checked
against on-disk code, `docs/packets/audits/STATUS.md`, and `git log`. The full verification
gate (Task 18) passed clean across all 5 changed Go modules (build/vet/test-race), all 3
`docker buildx bake` targets, redis/goroutine guards, the Go layer of `tools/lint.sh --check`,
and all three `packet-audit` `--check` gates. Working tree is clean; branch diff is 104 files /
+7102/-253, consistent with the plan's declared scope. No task was silently skipped, stubbed,
or deferred without an explicit, justified, in-scope resolution (Task 17's `build.go` grader
fix was an explicit user-approved in-scope tool change, not scope creep).

One process-hygiene gap, not a completion gap: **plan.md's 100 step-level checkboxes are all
still `- [ ]`** (0 are `- [x]`) despite every task being demonstrably done — the execution
process tracked completion via `.superpowers/sdd/task-N-report.md` rather than editing the
plan.md source file's checkboxes. This does not indicate missing work (see per-task evidence
below) but is worth flipping before merge so the committed plan.md reflects reality.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | `item_use.go` updateTime gate fix (`>=95`→`>=87`+JMS) | DONE | Discovered the `>=87` numeric fix was already on `main` (commit `18400caad`) with the `Region()=="GMS"` qualifier dropped; added the required exported `UpdateTimeFirst(t tenant.Model) bool` helper at `libs/atlas-packet/cash/serverbound/item_use.go` and replaced both gate sites. Commit `2451c4771`. Test `TestItemUseUpdateTimeFirst` (9 subtests, all versions) + full-package tests pass (`.superpowers/sdd/task-1-report.md:82-102`). |
| 2 | `ItemUseNote` arm codec + 9-version fixtures | DONE | New `libs/atlas-packet/cash/serverbound/item_use_note.go` (`NewItemUseNote`, `ToName/Message/UpdateTime`, `Encode/Decode`) + `item_use_note_test.go` (`TestItemUseNoteDecodeTrailing/Leading/RoundTrip`), verbatim per brief, all green. Commit `5bca92c64`. |
| 3 | Fix `NoteSendErrorBody` mode byte + `NO_NOTE_ITEM` key | DONE | `libs/atlas-packet/note/clientbound/operation_body.go`: `ResolveCode` call switched from `NoteOperationSendSuccess`→`NoteOperationSendError`; added `NoteSendErrorNoNoteItem = "NO_NOTE_ITEM"` const. Test `TestNoteSendErrorBodyUsesSendErrorMode` pins `[0x05,0x03]`/`[0x05,0x01]`. Commit `bc754d3b2`. |
| 4 | `libs/atlas-saga` — `NoteSend`/`CreateNote`/`CreateNotePayload` | DONE | `model.go` (`NoteSend Type`, `CreateNote Action`), `payloads.go` (`CreateNotePayload`), `unmarshal.go` (case). `TestCreateNoteStepUnmarshal` round-trips. Commit `db93f0033`. |
| 5 | atlas-notes — transaction id threading + `CREATE_FAILED` | DONE | `kafka/message/note/kafka.go` (`TransactionId` leading field on `Command`/`StatusEvent`, `StatusEventTypeCreateFailed`), `note/producer.go`/`processor.go` threaded, `kafka/consumer/note/consumer.go` no longer swallows the create error, `resource.go` passes `uuid.Nil`. `go test -race ./...` clean. Commit `2265f61bc`. Sanctioned deviation: producer import is the shared `atlas-kafka/producer`, not the brief's nonexistent local path (`.superpowers/sdd/task-5-report.md:84-87`). |
| 6 | Orchestrator — note message defs, processor + mock | DONE | New `kafka/message/note/kafka.go` (field-for-field JSON mirror of Task 5), `note/producer.go` (`CreateNoteCommandProvider`), `note/processor.go` (`Processor.CreateNote`), `note/mock/processor.go`. `TestCreateNoteCommandProvider` passes. Commit `2f7cfd1a5`. Same sanctioned shared-lib producer-path deviation as Task 5. |
| 7 | Orchestrator — `CreateNote` action wiring | DONE | `saga/model.go` re-exports + unmarshal case, `saga/handler.go` (`WithNoteProcessor`, `handleCreateNote` — correctly does NOT self-complete, mirroring async-completion actions), `saga/event_acceptance.go` (`EventKindNoteCreated`/`CreateFailed`, tables). `TestHandleCreateNote` (2 subtests) pass. Commit `52c6a374e` + follow-up commit `17d18199b` fixing a completeness-test gap (`allActions` list) caught in self-review — evidence the author actually re-verified rather than declaring done prematurely. |
| 8 | Orchestrator — note status consumer + main registration | DONE | New `kafka/consumer/note/consumer.go` (`handleCreatedEvent`/`handleCreateFailedEvent`, nil-txn guard for REST-created notes), registered in `main.go`. Commit `2f3a748b4`. One test (`TestHandleCreateFailedEvent_FailsCreateNoteStep`) was legitimately `t.Skip`'d pending Task 9's compensation wiring, per the plan's own explicit fallback instruction — verified removed and passing in Task 9 (see below). |
| 9 | Orchestrator — note_send completed Results + compensation | DONE | `saga/producer.go` (`extractNoteSendResults`, `NoteSend` branch in `CompletedStatusEventProvider`), `saga/compensator.go` (`DispatchNoteSendRollbacks`, `compensateNoteSend`, `NoteSend` branch in `CompensateFailedStep`). Task 8's skip removed and `TestHandleCreateFailedEvent_FailsCreateNoteStep` confirmed PASSING (`.superpowers/sdd/task-9-report.md:69-77`); `grep -n "t.Skip" consumer_test.go` → no matches. New tests `TestCompletedStatusEventProviderNoteSendResults`, `TestNoteSendCompensationRefundsItem`, `TestNoteSendCompensationDestroyFailedNoRefund` all pass. Commit `e547c262c`. |
| 10 | atlas-channel — saga message fields + note_send consumer branches | DONE | `kafka/message/saga/kafka.go` (`SagaTypeNoteSend`), `kafka/consumer/saga/consumer.go` (`handleCompletedEvent`/`handleFailedEvent` `note_send` branches, `extractResultCharacterId`). `TestExtractResultCharacterId` (4 cases) passes. Commit `f19a1ddae`. Sanctioned deviation: `StatusEventCompletedBody{SagaType,Results}` and `wp` param were already present from prior MTS work — only the `note_send` branch logic was added, as the parent brief pre-authorized. |
| 11 | atlas-channel — `FindFirstByClassification` + `buildNoteSendSaga` | DONE | `compartment/model.go` (`FindFirstByClassification`), new `socket/handler/note_send.go` (`buildNoteSendSaga` — `DestroyAsset` step 0 / `CreateNote` step 1, confirming FR-5 order; `handleNoteSendRequest`). Tests `TestFindFirstByClassification`, `TestBuildNoteSendSaga` pass. Commit `12dbf1643`. Author flagged (and later resolved in Task 12) a transient `unused` lint finding for `handleNoteSendRequest` pending its Task-12 call site — correctly not masked with `//nolint`. |
| 12 | atlas-channel — USE_CASH_ITEM note arm | DONE | `character_cash_item_use.go`: `CashSlotItemTypeNote = CashSlotItemType(21)`, `updateTimeFirst := cashsb.UpdateTimeFirst(t)` (Task 1 helper), new note arm calling `handleNoteSendRequest`. `TestCashSlotItemTypeNote`, `TestCharacterCashItemUseHandleFuncSymbol` pass; confirmed the Task-11 `unused` lint finding is now resolved (single call site). Commit `2ece631f6`. |
| 13 | atlas-channel — NOTE_ACTION SEND arm ownership gate | DONE | `note_operation.go`: SEND arm rewritten to `FindFirstByClassification(item.ClassificationNote)` pre-flight → `SEND_ERROR NO_NOTE_ITEM` on miss, else `handleNoteSendRequest`. Dead `SendNote`/`CreateCommandProvider` removed from `note/processor.go`/`producer.go`, confirmed via grep no other caller referenced them. `go build`/`vet`/`test -race` clean. Commit `e161a1ed4`. |
| 14 | Seed templates (9 versions) + live-tenant PATCH doc | DONE | All 9 `template_*.json` under `services/atlas-configurations/seed-data/templates/` verified/patched: `NO_NOTE_ITEM:3` added to all 9 writer `errors` tables (gms_87/95 previously missing the table entirely); `NoteOperationHandle` `validator`/`operations` added where absent (gms_87/95/jms_185); v48/v61 `SEND_ERROR=4` vs v72+ `SEND_ERROR=5` confirmed via `jq` assertions across all 9 files. `docs/tasks/task-137-note-item-consumption/live-tenant-patch.md` is a complete PATCH runbook. Commit `106fc9c69`. Sanctioned deviation: the writer `operations` tables were already correct on main; real gaps fixed were `errors.NO_NOTE_ITEM`, handler `options.operations`, and the missing gms_95 `validator`. |
| 15 | Matrix promotion: MEMO_RESULT × gms_v84 | DONE | STATUS.md line 70, gms_v84 column (`0x029`) = ✅ (confirmed live in current STATUS.md). IDA-verified `OnMemoResult @0xa70785`, all 4 mode arms (Display/SendSuccess/SendError/Refresh) decompiled and fixture-tested. Commit `d2bd0dcaf`. |
| 16 | Matrix promotion: MEMO_RESULT × jms_v185 | DONE | STATUS.md line 70, jms_v185 column (`0x026`, rightmost) = ✅ (confirmed live). IDA-verified `OnMemoResult @0xb0c6d0`, all 4 mode arms decompiled/fixtured. Commit `593724426`. |
| 17 | Matrix promotion: NoteOperationDiscard × jms_v185 + v48/v61/v72/v79 | DONE | STATUS.md lines 634 (op row) and 954 (sub-struct row): all 9 version columns = ✅ (confirmed live). jms_v185 required a genuine wire-codec fix (`specialCount`/`extra1`/`extra2` fields, `455ccccdb`) plus verification (`04b890eb0`). The 4 legacy versions were blocked by a pre-existing `packet-audit` matrix-tooling bug (op-row consumption swallowing the sub-struct's own report) — root-caused, two structural fixes empirically measured to over-flip 1,151+ unrelated cells and rejected, then fixed via a narrowly-scoped explicit allowlist (`legacyConsumedSiblingWriters` in `tools/packet-audit/internal/matrix/build.go`) gated additionally on the writer's own evidence grading `StateVerified`. `git diff` confirmed exactly the 4 intended cells flipped, nothing else. Commits `455ccccdb`, `04b890eb0`, `3b8eca39e`, `619bb66ea` (ToolSHA refresh). This tool change is in-scope per explicit user authorization (stated in the audit brief) and was empirically bounded before landing — not scope creep. |
| 18 | Full verification gate | DONE | All steps green: `go build/vet/test -race` × 5 modules; `docker buildx bake atlas-channel atlas-saga-orchestrator atlas-notes` × 3 clean; `redis-key-guard.sh`/`goroutine-guard.sh` exit 0; `tools/lint.sh --check` Go layer clean (one transient lock-contention flake on an untouched module, re-verified clean in isolation); `packet-audit matrix/fname-doc/operations --check` all exit 0; PRD §10's 13-item acceptance sweep fully cited. `.superpowers/sdd/task-18-report.md`. |

**Completion Rate:** 18/18 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Independent Verification Performed (this audit, not just report-reading)

- `git log --oneline b64ee7936..HEAD` — confirmed 25 commits matching the per-task narrative (1 commit per task, plus Task 7's completeness fix-up and Task 17's 4-commit sub-sequence).
- `git diff --stat b64ee7936..HEAD` — 104 files changed, +7102/-253; spot-checked the file list against the plan's declared "File Structure" table (Tasks 1–14's paths all present; `tools/packet-audit/internal/matrix/build.go` present, matching Task 17's tool fix).
- `docs/packets/audits/STATUS.md` grep: `MEMO_RESULT` row (line 70) shows ✅ in all 9 version columns, including the v84 (`0x029`) and jms_v185 (rightmost `0x026`) cells claimed by Tasks 15–16. `NoteOperationDiscard` appears twice (op row line 634, sub-struct row line 954) — both fully ✅ across all 9 columns, confirming Task 17's claim including the 4 legacy-version cells.
- `git status --short` — clean working tree, no stray edits.
- `git branch --show-current` — `task-137-note-item-consumption`, confirming operation stayed inside the correct worktree.
- Confirmed `services/atlas-ui` has zero diff on this branch (`git diff --stat b64ee7936..HEAD -- services/atlas-ui` → empty) and `tools/template-opcode-order-guard.sh` is genuinely absent from this worktree (`ls` → No such file or directory) — both caveats are real branch-freshness artifacts, not adherence failures, exactly as the audit brief characterized them.

## Skipped / Deferred Tasks

None. No task was SKIPPED, and no PARTIAL implementations were found. The two multi-step
internal deferrals that did occur were both properly closed out within the plan's own
task sequence, not left open at the end of the branch:

- **Task 8's `t.Skip`** on `TestHandleCreateFailedEvent_FailsCreateNoteStep` was explicitly
  sanctioned by the plan's own Step 2 fallback instruction ("If it fails here for a
  compensation-specific reason... mark it with `t.Skip`... Do not leave the skip in the final
  tree") and was removed with the test passing in Task 9, confirmed by direct grep
  (`t.Skip` → no matches) and a passing test run.
- **Task 7's `allActions` completeness-list gap**, caught by the author's own self-review
  (not by code review catching an actual defect left in the tree) and fixed in the same task's
  scope via a documented follow-up commit (`17d18199b`) before moving to Task 8.

## Process-Hygiene Note (not a completion gap)

`docs/tasks/task-137-note-item-consumption/plan.md` contains 100 step-level `- [ ]` checkboxes
(`Step 1`/`Step 2`/... under each of the 18 tasks) and **zero** are checked (`- [x]`), despite
every task being independently confirmed DONE above. The execution process
(`.superpowers/sdd/task-N-brief.md` / `task-N-report.md`) tracked completion out-of-band rather
than editing plan.md's own checkboxes. This is worth fixing before merge — a reviewer skimming
plan.md alone (without the `.superpowers/sdd/` reports) would see an entirely unchecked plan and
could misjudge the branch as unstarted — but it is a documentation-sync issue, not evidence that
any task was left undone; the code, tests, commits, and STATUS.md cells all corroborate
completion independently of the checkbox state.

## Build & Test Results

(From Task 18's gate, cross-referenced against this audit's independent STATUS.md/git checks;
not re-run in full during this audit per the parent's instruction that the gate already passed
and re-running the whole suite was unnecessary absent a specific doubt.)

| Module/Service | Build | Tests (-race) | Notes |
|---|---|---|---|
| libs/atlas-packet | PASS | PASS | Tasks 1, 2, 3, 15, 16, 17 |
| libs/atlas-saga | PASS | PASS | Task 4 |
| services/atlas-notes | PASS | PASS | Task 5 |
| services/atlas-saga-orchestrator | PASS | PASS | Tasks 6–9 |
| services/atlas-channel | PASS | PASS | Tasks 10–13 |
| docker buildx bake (atlas-channel, atlas-saga-orchestrator, atlas-notes) | PASS (all 3) | n/a | Task 18 |
| tools/redis-key-guard.sh | exit 0 | n/a | |
| tools/goroutine-guard.sh | exit 0 | n/a | |
| tools/lint.sh --check (Go layer) | PASS | n/a | one transient lock-contention flake on unrelated untouched module, re-verified clean |
| tools/lint.sh --check (UI layer) | N/A — environment gap | n/a | no Node 22 in shell; zero atlas-ui files touched on this branch (confirmed) |
| packet-audit matrix/fname-doc/operations --check | exit 0 (all 3) | n/a | |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Optional, cosmetic) Flip plan.md's 100 step-level checkboxes to `- [x]` (or add a summary
   note at the top of plan.md pointing to `.superpowers/sdd/task-*-report.md`) so the committed
   plan document reflects the branch's actual completion state for future readers who don't dig
   into `.superpowers/sdd/`.
2. No code, test, or verification-gate fixes are required. The branch is ready for the
   guideline-reviewer passes (`backend-guidelines-reviewer`) and PR creation.
