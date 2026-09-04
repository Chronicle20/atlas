# Plan Audit — task-285-maker-skill-crafting (Tasks 1-14)

**Plan Path:** docs/tasks/task-285-maker-skill-crafting/plan.md
**Audit Date:** 2026-09-01
**Branch:** task-285-maker-skill-crafting
**Base Branch:** main (diff base `9cd1ec5af`, HEAD `79f6bd566`)
**Scope:** Tasks 1-14 of 27 only. Tasks 15-27 (plus controller-inserted Tasks 26a/26b) are covered by a separate shard and are out of scope here.

## Executive Summary

All 14 tasks in this range are implemented, and the evidence trail (commits, code, `.superpowers/sdd/plan/progress.md` ledger, and mirrored review artifacts under `docs/tasks/task-285-maker-skill-crafting/reviews/`) is unusually thorough for this branch — every task went through an implementer + independent reviewer pass, several through fix rounds with re-review, and the controller ledger records rulings on every implementer deviation with evidence rather than taking self-reports at face value. All five affected Go modules (`atlas-data`, `libs/atlas-packet`, `libs/atlas-saga`, `atlas-inventory`, `atlas-saga-orchestrator`) build and test clean in this audit's own re-run. The one gap found is a documentation-durability issue, not a code defect: the review artifacts for Tasks 3, 12, 13 and 14 were never committed to `docs/tasks/.../reviews/` (they exist only as APPROVED verdicts recorded inline in the gitignored `progress.md` ledger, and for 12-14 also as gitignored `.superpowers/sdd/plan/task-N-review.md` files) — the same "gitignored reports are not durable" defect the ledger itself flags as an open item for the branch wrap-up.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Registry correction — `MAKER_SKILL` on `gms_v72`/`gms_v79` | DONE | `docs/packets/registry/gms_v72.yaml:2647-2652`, `gms_v79.yaml:3143-3149` add the serverbound entry (opcode 112/111, `fname: CUIItemMaker::RequestItemMake`). `docs/packets/audits/STATUS.md:655` shows `MAKER_SKILL` ✅ on all eight applicable versions after later tasks pin it. Review `reviews/task-1.md` APPROVED, 0 blocking. |
| 2 | `atlas-data` `itemmake` — RestModel and registry | DONE | `services/atlas-data/atlas.com/data/itemmake/rest.go`, `registry.go` present; `go test ./itemmake/... -count=1` passes (this audit's own run). Review `reviews/task-2.md` APPROVED, 0 blocking; confirmed `Group` (C-6) and `ReqQuest` (C-5) fields present, no GORM entity/migration (C-4). |
| 3 | `atlas-data` `itemmake` — the archive reader | DONE | `services/atlas-data/atlas.com/data/itemmake/reader.go`, `reader_test.go` present (commit `f6d91735e`). Review recorded APPROVED (0 blocking, 1 non-blocking coverage gap deferred) only in `progress.md:143-160` — **the standalone `reviews/task-3.md` artifact was never committed**, unlike every sibling task's review. |
| 4 | `atlas-data` `itemmake` — processor, storage, REST resource | DONE | `services/atlas-data/atlas.com/data/itemmake/processor.go`, `resource.go`, `mock/processor.go` present; `main.go` wired via one `AddRouteInitializer` line (commit `cda477ff5`, fix round `5a3cdb3ed` for errcheck). Review `reviews/task-4.md` APPROVED, 0 blocking. |
| 5 | `atlas-data` — `ITEM_MAKE` worker registration | DONE | `WorkerItemMake` registered in `services/atlas-data/atlas.com/data/processor.go`; commit `2df70429f` + fix round `6ff166527` (test strengthened from vacuous nil-check to a real persisted-row assertion after review found it blocking). Review `reviews/task-5.md` APPROVED_WITH_FINDINGS → fix → `reviews/task-5-fix-1.md` ADDRESSED. |
| 6 | Per-version wire derivation, all eight versions | DONE | `docs/tasks/task-285-maker-skill-crafting/wire-derivation.md` present, with `MAKER_SKILL` and `MAKER_RESULT` both re-derived from live IDBs across two rounds (commits `2f2361700`, `73939663c`, fix `e29058143`). Review CHANGES_REQUIRED (untraceable `MAKER_RESULT` claims) → fix round quoted per-version disassembly → `reviews/task-6-fix-1.md` ADDRESSED. |
| 7 | `MAKER_SKILL` serverbound codec | DONE | `libs/atlas-packet/character/serverbound/maker_skill.go` + `maker_skill_test.go` present; `go build`/`go test` pass in this audit's own module run. Commit `ed190de3a`, fix round `b19fe1a92` (stale `STATUS.md`/`status.json` from the `toolSha` trap). Codec itself confirmed correct by review (`reviews/task-7.md`); fix scoped to the audit artifacts only. Evidence pinning for the op landed separately (`640b8caa1`, `reviews/task-7-pinning.md` APPROVED_WITH_FINDINGS) — `MAKER_SKILL` reads ✅ on all eight versions in `docs/packets/audits/STATUS.md:655`. |
| 8 | `MAKER_RESULT` dispatcher family — arm structs and body functions | DONE | `libs/atlas-packet/character/clientbound/maker_result.go` (395 lines, five arm structs) + `maker_result_body.go` present. Commit `8f596fd97`. Review returned CHANGES_REQUIRED but was dismissed by controller ruling as a scope error (fixtures/round-trip tests are Task 9's scope per the plan's own split, mirrored to `ruling-failed-arm.md`); no fix round required for Task 8 itself. `FAILED` arm's `WithResolvedCode` omission ruled ACCEPTED against `resolve.go`/`dispatcher_lint.go` and the `guild.Info` precedent. |
| 9 | `MAKER_RESULT` — dispatcher YAML, operations tables, byte fixtures | DONE | Commit `ab31624bd` + fix round `5d2a2ed52`. `docs/packets/audits/STATUS.md:334` shows `MAKER_RESULT` ✅ on all eight applicable versions. First-round review found the ✅ was evidence-backed for only one of four non-degenerate arms (`CreateWithUpgrade`, `MonsterCrystal`, `Disassemble` had fixtures but no marker/evidence) — CHANGES_REQUIRED, 1 blocking. Fix round generated per-arm audit reports for all four arms and was independently confirmed by a negative-control test (deleting one arm's evidence file flips its cell ✅→❌ and back) — `reviews/task-9-fix1.md` APPROVED, 0 blocking. `FAILED` key correctly omitted from the operations table per user ruling. |
| 10 | `libs/atlas-saga` — `AwardCraftedAsset` action | DONE | `libs/atlas-saga/model.go:251-257` (`Action` constant), `payloads.go:1069-1078` (`AwardCraftedAssetPayload`), `unmarshal.go:486-487` (`Step[T].UnmarshalJSON` case), `payloads_test.go:87-156` (4 tests incl. `Slots` zero-value survival). `go test ./...` passes in this audit's own run. Commit `3ba8a3aff`. Review APPROVED, 0 blocking (reconstructed into `reviews/task-10.md` after the reviewer failed to write its own durable artifact — noted in `progress.md`). |
| 11 | `atlas-inventory` — explicit-stat asset creation | DONE | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:39-73` (`EquipStats` struct, `firstEquipStats` helper), trailing-variadic `stats ...EquipStats` added to `CreateAssetAndEmit`/`CreateAssetAndLock`/`CreateAsset` (`:116-118`, `:1283`). `go build`/`go test ./...` pass in this audit's own run. Commit `b36fff470`. Review `reviews/task-11.md` APPROVED_WITH_FINDINGS, 0 blocking — confirmed the seam test (`TestCreateAssetAppliesExplicitStats` et al.) asserts on the persisted `asset.Model`, not just the options struct, and the variadic-vs-pointer, `compartment/mock/processor.go` fallout, and `omitempty` asymmetry were all ruled non-defects with evidence. |
| 12 | `atlas-saga-orchestrator` — explicit-stat creation command | DONE | Producer-side `CreateAssetCommandBody` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/.../kafka.go:135-149` confirmed character-for-character matching the inventory-side copy (`kafka.go:117-132`) per review. Commit `7b07f6ee2`. Review APPROVED, 0 blocking/0 non-blocking/0 not-evaluable — recorded only in `progress.md` and the gitignored `.superpowers/sdd/plan/task-12-review.md`; **no `reviews/task-12.md` was committed to `docs/tasks/.../reviews/`.** |
| 13 | `atlas-saga-orchestrator` — `AwardCraftedAsset` handler and step completion | DONE | `saga/compensator.go:61` interface method, `:532-533` dispatch case (handler wiring); `saga/model.go` both changes present (aliases + local `Step[T].UnmarshalJSON` arm per plan's explicit two-change note); `event_acceptance.go` pair reuses `AwardAsset`'s event kinds. `saga/rest.go` deliberately untouched per plan. Commit `c96dd1f`. Review APPROVED, 0 blocking/0 non-blocking/2 not-evaluable — recorded only in `progress.md` and gitignored `task-13-review.md`; **no `reviews/task-13.md` committed.** |
| 14 | `atlas-saga-orchestrator` — `AwardCraftedAsset` compensation | DONE | `saga/compensator.go:1185-1287` (`compensateAwardCraftedAsset`, reverse-walk through the saga-type-agnostic action switch, modelled on `compensateSelectGachaponReward`), `:3134-3137` (late-compensation pairing entry), `:3288-3292` (dispatch arm). `TestCraftSagaFullyCompensatesOnFinalStepFailure` (`compensator_test.go:1379-1478`) asserts the FR-3.7 no-partial-compensation invariant. Commit `a1d635c`. Review APPROVED_WITH_FINDINGS, 0 blocking/1 non-blocking (untested `CompensateFailedStep`-entry dispatch arm, low risk, carried forward not fixed) — recorded only in `progress.md` and gitignored `task-14-review.md`; **no `reviews/task-14.md` committed.** |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 14 tasks in this range have code evidence, passing builds/tests, and a recorded reviewer verdict of APPROVED or APPROVED_WITH_FINDINGS (Task 8's initial CHANGES_REQUIRED was dismissed by an evidenced controller ruling as a scope-boundary error against the plan's own Task 8/9 split, not a defect in Task 8's work; Task 9's initial CHANGES_REQUIRED was fixed and re-reviewed to APPROVED).

The only gap found is a **documentation-durability gap, not an implementation gap**: review artifacts for Tasks 3, 12, 13 and 14 were never mirrored into the committed `docs/tasks/task-285-maker-skill-crafting/reviews/` directory (present for 1, 2, 4, 5, 6, 7, 8, 9, 10, 11, but absent for 3, 12, 13, 14). Their APPROVED verdicts and findings survive only in `progress.md` (gitignored) and, for 12-14, in `.superpowers/sdd/plan/task-N-review.md` (also gitignored). This is the exact "gitignored reports are not durable" defect the branch's own ledger flags as an open item to close before wrap-up (`progress.md` open item 2). No code impact — implementation evidence for all four tasks is present and independently verifiable in the diff (see table above).

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data (services/atlas-data/atlas.com/data) | PASS | PASS | `go test ./itemmake/... -count=1` — all green (Tasks 2-5) |
| libs/atlas-packet | PASS | PASS | `go test ./character/... -count=1` — all green (Tasks 7-8) |
| libs/atlas-saga | PASS | PASS | `go test ./... -count=1` — all green (Task 10) |
| atlas-inventory (services/atlas-inventory/atlas.com/inventory) | PASS | PASS | `go test ./... -count=1` — all green (Task 11) |
| atlas-saga-orchestrator (services/atlas-saga-orchestrator/atlas.com/saga-orchestrator) | PASS | PASS | `go test ./... -count=1` — all green (Tasks 12-14) |

All five results are from this audit's own re-run against the worktree at HEAD `79f6bd566`, not taken from implementer/reviewer reports. `tools/verify.sh` was not run per the task instructions (a flagless run was already in flight in this session).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; the branch's overall merge readiness also depends on the Tasks 15-27 shard's findings, which are out of scope here)

## Action Items

1. (Non-blocking, cleanup) Mirror `reviews/task-3.md`, `reviews/task-12.md`, `reviews/task-13.md`, `reviews/task-14.md` from `progress.md` / the gitignored `.superpowers/sdd/plan/task-N-review.md` files into `docs/tasks/task-285-maker-skill-crafting/reviews/`, consistent with every other task in this range and with the branch's own tracked open item 2 in `progress.md`. This does not block merge but should happen before the branch's final documentation sweep.
