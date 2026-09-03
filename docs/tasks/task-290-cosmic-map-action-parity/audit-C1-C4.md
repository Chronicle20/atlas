# Plan Audit — task-290-cosmic-map-action-parity (Tasks C1-C4)

**Plan Path:** docs/tasks/task-290-cosmic-map-action-parity/plan-c.md
**Audit Date:** 2026-09-03
**Branch:** task-290-cosmic-map-action-parity
**Base Branch:** main (Plan C base commit: 1af60de4f)

## Executive Summary

All four tasks in Plan C's first shard (C1–C4) are fully implemented, matching the plan's
prescribed code and seed-document text verbatim in every case checked. Each task's commit
is separately gated (`tools/verify.sh --quick`) and reviewed (`task-reviewer`, all
APPROVED or APPROVED_WITH_FINDINGS with zero blocking findings). Module builds and tests
pass for `atlas-map-actions`, `libs/atlas-saga`, and `atlas-saga-orchestrator`; all 88 seed
documents (8 filenames × 11 roots) are present, byte-identical across roots, and pass
`gen-map-action-schema.sh --check` and `catalog-lint` with a clean `deploy/seed/` tree.
Task C4 surfaced one genuine cross-service gap (the client is never told a cleared skill
was removed) which the controller correctly declined to silently absorb into this task
and instead filed as a written follow-up — that follow-up is present on disk and is the
right disposition, not a plan-adherence defect.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| C1 | `set_quest_progress` and `start_quest` executor arms (G3, G4) | DONE | `executor.go` switch cases + `executeSetQuestProgress`/`executeStartQuest` match plan verbatim (commit f27aa19b4). Tests `TestExecuteSetQuestProgress(ParamValidation)`, `TestExecuteStartQuest(DefaultsNpcIdToZero/ParamValidation)` all present at executor_test.go:345-483+. Schema `allOf` blocks added, `gen-map-action-schema.sh --check` exit 0. 4 seed docs × 11 roots = 44 files, byte-identical across roots, content matches plan §Step 7 exactly (incl. `map-babyPigMap.json`'s `unlock_ui` before `start_quest` ordering). |
| C2 | `open_npc` executor arm (G9) | DONE | `executeOpenNpc` present with the plan's exact AccountId=0 rationale comment (commit 84035b8df). Step 1's investigation was done for real: report and ledger record that `handleStartNpcConversation` DOES read `AccountId` but the map-enter command carries none — a genuine finding, not a guess, and traced independently by the reviewer to `atlas-npc-conversations/conversation/processor.go:176-178`. Tests `TestExecuteOpenNpc`/`TestExecuteOpenNpcParamValidation` present. 2 seed docs × 11 roots = 22 files, content matches plan (`Resi_tutor40`→2159012, `Resi_tutor60`→2159007); `Resi_tutor50` correctly left unconverted per §0. |
| C3 | `show_info` and `update_area_info` executor arms (G12 partial) | DONE | `executeUpdateAreaInfo`/`executeShowInfo` match plan verbatim, including `uint16` parse width for `area` (commit 87452ba2d). Tests `TestExecuteUpdateAreaInfo`, `TestExecuteShowInfo`, `TestExecuteAreaInfoParamValidation` present at executor_test.go:655-738+. 1 seed doc × 11 roots = 11 files, `map-Resi_tutor30.json` content matches plan exactly, operation order (`update_area_info` then `show_info`) matches Cosmic's call order. |
| C4 | `clear_skill` executor arm and `iceCave` (G13) | DONE | Full 7-touchpoint saga action wiring confirmed by direct diff read: `libs/atlas-saga/{model,payloads,unmarshal}.go`, orchestrator `saga/{model,event_acceptance,handler,character_extractor}.go`, `skill/processor.go` doc-comment update (no new method — reused `RequestDeleteSkill`, per the Step 1 investigation finding that a removal path already existed downstream). Two documented deviations from the plan's literal sketch (`WorldId` on payload, `EventKindSkillDeleted` acceptance instead of `{}`) are both correctly justified and independently reviewer-confirmed. `TestUnmarshalClearSkillStep` and `TestExecuteClearSkill(ParamValidation)` present. `event_acceptance_test.go`'s `allActions` list extended with `sharedsaga.ClearSkill`. `map-iceCave.json` × 11 roots = 11 files, 7-operation sequence (5× `clear_skill`, `unlock_ui`, `show_intro`) matches plan exactly. |

**Completion Rate:** 4/4 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All four tasks are fully implemented per plan.

One item worth flagging as context rather than a deficiency: Task C4's implementation
correctly identified that Cosmic's `updateSkill(id,-1,0,-1)` clientbound notification has
no Atlas equivalent on the `clear_skill` path (`atlas-channel`'s
`handleSnapshotSkillDeleted` discards its packet writer). This is **not** an unmet C4
acceptance criterion — the plan's own Files list for C4 never named atlas-channel or any
packet writer, and the plan's Global Constraints explicitly fence off "No new packets" /
new opcode work as out of scope for this plan, routing it to
`docs/packets/PROCESS.md`. The controller filed
`docs/tasks/task-290-cosmic-map-action-parity/followup-clear-skill-wire-notification.md`
recording the gap with full trace evidence, and surfaced it to the user per the ledger.
That is the correct disposition under CLAUDE.md's "Finish producible work" rule, since the
missing piece (a level-`-1` clientbound skill-change encoding) requires client-binary
derivation this plan does not have and is not a prerequisite the implementer could
"just produce."

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| services/atlas-map-actions/atlas.com/map-actions | PASS | PASS | `go build ./...` and `go test ./... -count=1` both green; `script` package covers all new arms. |
| libs/atlas-saga | PASS | PASS | `go build ./...` and `go test ./... -count=1` green, including `TestUnmarshalClearSkillStep`. |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | `go build ./...` and `go test ./... -count=1` green across all packages, including `saga` and `saga/mock`. |
| schema / seed | PASS | PASS | `./tools/gen-map-action-schema.sh --check` exit 0; `catalog-lint` exit 0 over `deploy/seed`; `git status --short deploy/seed/` empty (0 dirty files) after re-running the loop. |

Per-task `tools/verify.sh --quick` gates recorded in `.superpowers/sdd/plan-c/progress.md`
all report exit 0 (C1 over 1af60de4f..f27aa19b4, C2 over f27aa19b4..84035b8df, C3 over
84035b8df..87452ba2d, C4 over 87452ba2d..4278bded9). This audit did not re-run the full
(non-`--quick`) `tools/verify.sh`, which is scoped to the whole-branch final gate rather
than a per-shard audit; the module-level build/test results above are a direct
re-verification of the shard's Go surface.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this shard; whole-branch merge readiness is a
  question for the final Plan C gate covering C1-C23, not this range alone)

## Action Items

None required for C1-C4. The one open item — clientbound notification for a cleared
skill — is correctly scoped out of this plan and already tracked in
`followup-clear-skill-wire-notification.md`; no action needed within this shard's task
range.
