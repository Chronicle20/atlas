# Plan Audit — task-254-party-experience-sharing (Tasks 8-13)

**Plan Path:** docs/tasks/task-254-party-experience-sharing/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-254-party-experience-sharing
**Base Branch:** main (merge base d17404dbc)
**Scope:** Plan Tasks 8-13 only (tasks 1-7 audited separately)

## Executive Summary

All six plan tasks in this range (8-13) are fully implemented and match the plan's Interfaces/Files/algorithm specifications almost line-for-line. `ExperienceConfig` and its env loader (Task 8), the plan value types and `computeAward` (Task 9), `planDistribution` (Task 10), `ProcessorImpl` dependency injection (Task 11), the resolve→plan→award rewrite of `DistributeExperience` with retirement of `produceDistribution`/`distributeCharacterExperience`/`DamageDistributionModel` (Task 12), and the mock-driven I/O test suite (Task 13) are all present with matching test names and passing tests. Both noted fix rounds are confirmed: the overlay `configMapGenerator` keys were re-listed in `main`/`pr` kustomizations (commit `e774a09bb`), and the bare `go` statements in the monster-info/field fan-out were converted to `routine.Go` (commit `21df4197c`). `go build ./...` and `go test ./... -race -count=1` both pass cleanly for the module. The out-of-brief deletion of `monster/builder.go` and `monster/builder_test.go` is confirmed complete — those files were the `DamageDistributionBuilder` test-construction helper for the now-deleted `DamageDistributionModel`, and no reference to any of the three retired symbols (`produceDistribution`, `distributeCharacterExperience`, `DamageDistributionModel`) remains anywhere in the service.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 8 | `ExperienceConfig` and its env loader | DONE | `monster/config.go` matches plan's struct/defaults/loader verbatim; `monster/config_test.go` has `TestDefaultExperienceConfig` and `TestLoadExperienceConfig` (table-driven, matches plan's 8 cases). `deploy/k8s/base/env-configmap.yaml` gained `LEACH_INTERVAL`, `LEVEL_INTERVAL`, `USE_ENFORCE_MOB_LEVEL_RANGE` at the specified positions (commit `b1a00f68f`). Fix round confirmed: `e774a09bb` re-lists the three keys in `deploy/k8s/overlays/main/kustomization.yaml` and `deploy/k8s/overlays/pr/kustomization.yaml`, whose `configMapGenerator` uses `behavior: replace`. |
| 9 | Plan value types, `computeAward`, hint text | DONE | `monster/experience.go:9-143` defines `DamageInput`, `SoloInput`, `MemberInput`, `PartyInput`, `ExperienceInput`, `Recipient`, `Exclusion`, `ExperiencePlan`, `computeAward`, `toAwardAmount`, `levelGateHintText` — field-for-field match to the plan's code block. `monster/experience_test.go` has `TestComputeAward` (table), `TestComputeAward_NonFiniteIsGuarded`, `TestLevelGateHintText`. Reviewer `review-task-9.md` verdict: APPROVED, 0 blocking/non-blocking. |
| 10 | `planDistribution` | DONE | `monster/experience.go:146-318` implements the 11-step algorithm exactly as specified (sort copies, `damageOf`/`totalDamage`, zero-damage short-circuit, `epd`, entry-composition counting, `personalRatio`/`entryRatios`, sorted-then-`calculateExperienceStandardDeviationThreshold`, solo recipients, per-party contributor/gate/MVP/bonus logic, final sort). Tests present: `TestPlanDistribution` (all listed rows), `TestPlanDistribution_PartiesOrderedByPartyIdMembersByCharacterId`, `TestPlanDistribution_IsDeterministicUnderShuffledInput`, `TestPlanDistribution_TotalEntriesComposition`. `processor.go`'s `calculateExperienceStandardDeviationThreshold`/`isWhiteExperienceGain` untouched (D4). Reviewer noted one non-blocking asymmetry in the `inFieldDamagers` counting (solo counted by presence, party member by `d>0`) — unexercised by tests but functionally harmless per reviewer's analysis; not a plan-adherence gap. |
| 11 | Inject collaborators into `monster.ProcessorImpl` | DONE | `monster/processor.go:31-113` adds all 8 fields (`cp,pp,rp,ip,fp,smp,ht,cfg`), binds production defaults in `NewProcessor`, adds `ProcessorOption` + 8 `With*` functions + `With(...)`, matching the plan's code block exactly (including the `clone := *p; cp := &clone` pattern). `CreateDrops` routed through `p.rp`/`p.pp`. `monster/processor_di_test.go` has `TestWith_ReturnsCloneAndDoesNotMutateOriginal`, `TestWith_AppliesEveryOption`, `TestNewProcessor_BindsProductionDefaults`. Reviewer verdict: APPROVED, no findings. |
| 12 | Rewrite `DistributeExperience` as resolve→plan→award | DONE | `monster/processor.go:210-350` implements resolve (concurrent `p.ip.GetById`/`p.fp.CharacterIdsInField` via `routine.Go`+`sync.WaitGroup`), party resolution with solo fallback on error, `planDistribution` call, award loop with per-recipient `continue`-on-error, and hint loop with throttle gate — matching the plan's pseudocode and prose precisely. `produceDistribution`, `distributeCharacterExperience`, `DamageDistributionModel` are fully deleted; `grep -rn 'produceDistribution\|distributeCharacterExperience\|DamageDistributionModel' services/atlas-monster-death/atlas.com/monster` returns nothing. `model.go` retains only `DamageEntryModel`/`NewDamageEntryModel`/its two accessors. `grep -rn 'TODO parties\|TODO account for healing'` and `grep -n 'totalPartyLevel byte'` both print nothing, confirming the final-gate greps pass. `aggregateDamageEntries` present at `experience.go:19-36` with `TestAggregateDamageEntries` covering all 6 table rows. Both out-of-brief deletions (`monster/builder.go`, `monster/builder_test.go`) confirmed: `git show d17404dbc:.../monster/builder.go` shows `DamageDistributionBuilder`, the test-construction helper for `DamageDistributionModel`; `grep -rln 'monster\.NewBuilder\|monster\.Builder' services/atlas-monster-death/` finds zero remaining callers repo-wide. Fix round confirmed: commit `21df4197c` converts the two bare `go func(){...}()` statements at the old `processor.go:230-236` to `routine.Go(p.l, p.ctx, func(_ context.Context) {...})`, present unchanged in current `processor.go:231-238`. Reviewer verdict: APPROVED_WITH_FINDINGS (0 blocking, 2 non-blocking: a WARN-log-noise observation on the "no party" vs "outage" ambiguity, both attributed to the brief's own contract, not an implementer deviation). |
| 13 | Processor tests with mocks | DONE | `monster/processor_experience_test.go` contains all 15 tests named in the plan's table verbatim: `TestDistributeExperience_OnePartyLookupPerParty`, `_OneRateLookupPerRecipient`, `_PartyLookupErrorFallsBackToSolo`, `_PartyRecipientsCarryNonZeroParty`, `_SoloRecipientCarriesZeroParty`, `_ZeroDamageAwardsNothing`, `_OutOfFieldDamagerReceivesNothing`, `_ExcludedMemberGetsExactlyOneHint`, `_HintFailureDoesNotAbortAwards`, `_AwardFailureDoesNotAbortOthers`, `_HintIsThrottledAcrossKills`, `_GateDisabledEmitsNoHint`, `_InformationErrorReturnsError`, `_FieldErrorReturnsError`, `_AwardOrderIsAscendingCharacterId`. `go test ./... -race -count=1` passes with no race reports. Reviewer verdict: no blocking findings. |

**Completion Rate:** 6/6 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All six tasks in this range (8-13) have full evidence of implementation matching the plan's specified interfaces, files, and algorithms, and all corresponding tests are present and passing.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-monster-death (module `services/atlas-monster-death/atlas.com/monster`) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all packages `ok`; `go test ./... -race -count=1` all packages `ok`, no race reports. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; final merge readiness also depends on the other shard's audit of tasks 1-7 and the plan's Final Gate items, which are outside this shard's scope)

## Action Items

None required for tasks 8-13. The two non-blocking reviewer observations (Task 10's `inFieldDamagers` counting asymmetry, Task 12's WARN-log noise on routine "no party" lookups) are noted for awareness but do not block this range.
