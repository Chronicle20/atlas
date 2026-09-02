# Plan Audit — task-283-race-index-job-mapping

**Plan Path:** docs/tasks/task-283-race-index-job-mapping/plan.md
**Audit Date:** 2026-08-28
**Branch:** task-283-race-index-job-mapping
**Base Branch:** main (base commit 9cd1ec5af)
**HEAD:** 7a2275acb

## Executive Summary

All 9 plan tasks were faithfully executed, in the order the controller directed (Task 3 before Task 2's dependents, out-of-order but explicitly sanctioned). Every FR/NFR in the PRD is satisfiable from code+test evidence I read directly, findings.md is a real IDA-derived evidence document (not inference), and all three deliberate mid-execution corrections (R-2 grep note, the jms_185 exemption, the v95 minorVersion=1 fix, Task 8's finer split) are present in the code exactly as ruled. `go build`/`go test` pass clean in both `atlas-character-factory` and `atlas-constants`; `npm run build` and `npm test` pass clean (2356/2356) in `atlas-ui`. The only gap found is a plan-hygiene one, not a functional one: only 4 of 58 step-level checkboxes in `plan.md` are checked (Task 9's self-check steps); the other 54 steps across Tasks 1–8 were verifiably executed (confirmed against git history and diffs) but never marked `[x]`.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Derive every version's race carousel from client binaries | DONE | `findings.md` (398 lines) cites function+address per version, records `gms_12` unverified, `gms_v72/61/48` as "no race field," and resolves all three open questions (Resistance selectable, Dual Blade, jms_185 divergence) with citations. Commit `1df238793..c56d7f9fb` (plus a labelling fix-up `c56d7f9fb`, in scope). |
| 2 | Project findings into `docs/packets/race-carousels.json` | DONE | `docs/packets/race-carousels.json` has all 11 version keys, slot counts match `findings.md` row counts; two legitimate self-corrections (`0e1af6bf5` fixing the `verified` flag on gms_v79/2.0, `2cccecab7` restoring the frozen `jobId` per FR-21 while keeping `verified:false`) both landed inside the commit range attributed to Task 2/3, both cited. |
| 3 | Freeze pre-Big-Bang behavior, kill tautological assertions | DONE | `processor_test.go:400,1140` now use `const expectedJobId = job.BeginnerId` literals with cited reasoning; `TestFromIndex_PreBigBangFrozen` added at line 961, table-driven, later (Task 5) rewritten to call `job2.FromIndex` with an `ok` assertion — confirmed at `processor_test.go:1009-1023`. |
| 4 | New job constants — Citizen id, beginner set | DONE | `job.CitizenId = Id(3000)` added at `libs/atlas-constants/job/constants.go`, registered in `Jobs`, added to `IsBeginner`'s `IsA(...)` list, `advancement_test.go`/`constants_test.go` updated (len(Jobs) 82→83). Comment correctly flags the value as human-supplied, not IDA-derived, matching `findings.md`'s own citation. |
| 5 | Version-aware carousel mapper | DONE | `job/carousel.go` implements `Slot`/`Carousel`/`carouselFor`/`FromIndex` exactly per plan skeleton, six carousels (`unverifiedCarousel`, `noRaceCarousel`, `race3Carousel`, `race4Carousel`, `race5Carousel`, `race4JmsCarousel`), predicate chain uses only `IsRegion`/`MajorAtLeast`/`MajorInRange`. `libs/atlas-constants/job/model.go`'s `FromIndex` deleted. `carousel_test.go` has all four required tests: per-version table (a), off-carousel rejection incl. structural sweep (b), parity-fixture round-trip (c), multi-tenancy order-independence (d). |
| 6 | Wire mapper into creation, real rejection | DONE | `ErrInvalidRaceIndex` sentinel added; `Create` resolves `job2.FromIndex(t, ...)` once right after `tenant.MustFromContext`, rejects with the sentinel and a log line matching the required fields; `buildCharacterCreationSaga` gained a `jobId job.Id` parameter, all 10+ call sites updated; `validJob` stub and `job/model.go` (the service-local `JobFromIndex` twin) both deleted; `categorizeError` rewritten to an `errors.Is` switch ahead of the substring fallback, `"must provide valid job index"` removed from that list. `TestCreate_RejectsOffCarouselRaceIndex`, `TestCreate_ValidatorOrderIsPreserved`, `TestCategorizeError_InvalidRaceIndexIsBadRequest` all present and match the plan's exact table shapes. |
| 7 | Seed templates + carousel↔template correspondence gate | DONE | `template_gms_48/61/72_1.json`'s unreachable `(0,0)`/`(2,0)` rows deleted per the FR-8 positive contradiction; `template_gms_95_1.json` repermuted onto the verified v95 ordinals (Resistance at 0, Cygnus at 2, Aran at 3), missing `(4,0)` Evan row added, both human-supplied `mapId`s (Citizen 310010000, job id 3000) cited as non-IDA. `job/correspondence_test.go` implements both directions of the FR-19/FR-20 gate with cited, narrowly-scoped exemption maps (`direction1Exempt` = gms_12 only; `direction2Exempt` = gms_v92 (1,1) and all 5 gms_jms_185 slots, each with a `findings.md` citation) — matches the R-2/jms exemption ruling: `race4JmsCarousel` is genuinely empty (no entries) and all 5 jms slots carry `jobId: null` in the fixture, so leaving those seed rows untouched is correct, not a skip. Also confirmed FR-20's v84/v87 `mapId` divergence is explicitly *not* touched (`findings.md` says the client gives no basis to adjudicate it, and the plan's diff shows those two files untouched). |
| 8 | Version-aware class labels in the template editor | DONE | `jobNames.ts` rewritten: `classesForVersion`/`worldNameFromJobIndex`/`templateLabels` all take `(region, majorVersion)`; the stale "mirrors JobFromIndex" header comment is gone. `IdentitySection.tsx`, `TemplateSelector.tsx`, `CharacterTemplatesEditor.tsx` all thread `activeTenant?.attributes.region`/`.majorVersion` through, matching the documented `Tenant` shape. `raceCarousels.parity.test.ts` (new) reads the fixture from disk and round-trips every version key against `classesForVersion`. Two fix-round commits (`a821f0c43`, `b3c301eec`) landed within the plan's attributed range and are consistent with the "Task 8 splits Go's carousel ranges one level finer" ruling — `classesForVersion` has explicit, cited arms for the 79/83 anchor split and the 80–82 interpolation gap, correctly deferring to the v83 anchor labels rather than v79's geometry-only guess. |
| 9 | Full verification gate | DONE (build/test verified by me; `tools/verify.sh` intentionally NOT run per instructions — owned by the controller) | FR-4 grep (`FromIndex`) shows hits only in `carousel.go`, `carousel_test.go`, `processor.go`, `processor_test.go` — no `libs/atlas-constants` hit, no `JobFromIndex` code reference (only a comment). FR-2 raw-comparison grep and TODO/FIXME grep both empty. `TestGms95SeedStartMapsMatchFindings` (added in the final commit `7a2275acb`) closes the "v95 start-map mapIds asserted on inspection alone" gap the task description flagged. `plan.md`'s own Task-9 self-check steps 1/2/3/5 are checked `[x]`; step 4 (`tools/verify.sh`) and step 6 (gate-driven commit) correctly left unchecked/deferred to the controller. |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every FR/NFR I checked against code has direct evidence. The controller-supplied rulings (R-2, the jms_185 exemption, gms_v95 minorVersion=1, Task 8's finer split) are all correctly reflected in the current code, not merely asserted in a commit message.

One item worth surfacing as a **non-blocking** finding: `plan.md`'s per-step checkboxes were not updated as work progressed. Only 4 of 58 `- [ ]` step items are checked (Task 9 steps 1/2/3/5); Tasks 1–8's ~50 individual steps remain `[ ]` despite every one being independently confirmable as done from the git history and current file contents. This does not affect the delivered code — I traced every task's actual commits and file states directly rather than trusting the checkbox state — but it means `plan.md` alone understates completion and a future reader skimming only the checkboxes would misjudge the branch as far less complete than it is.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-character-factory (`services/atlas-character-factory/atlas.com/character-factory`) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all packages `ok` (`factory`, `job`, `configuration`, `data`, `kafka/consumer/saga`, etc.) |
| atlas-constants (`libs/atlas-constants`) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all packages `ok`, including `job` (CitizenId additions) |
| atlas-ui (`services/atlas-ui`) | PASS | PASS | `npm run build` (tsc -b + vite build) clean, only a pre-existing unrelated chunk-size warning; `npm test` → 280 test files / 2356 tests, all passed |

`tools/verify.sh` was intentionally not run — per instructions, the controller owns that gate and is running it separately.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the controller's own `tools/verify.sh` run and code review, both outside this audit's scope)

## Action Items

1. (Non-blocking, documentation only) Update `plan.md`'s step-level checkboxes for Tasks 1–8 to `[x]` to match the branch's actual completion state, so the plan document itself is an accurate record rather than requiring a git-history reconstruction to confirm.
