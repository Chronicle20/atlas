# Code Review Audit — task-121-local-map-membership

**Date:** 2026-07-02
**Branch:** task-121-local-map-membership · **Base:** main
**Implementation range:** 008396e97..e3903d817 (7 commits; plan committed at 008396e97)

## Verdicts

| Review | Verdict |
|--------|---------|
| Whole-branch review (Opus) | Ready to merge — no Critical/Important/Minor |
| plan-adherence-reviewer | READY_TO_MERGE — 6/6 tasks implemented |
| backend-guidelines-reviewer | PASS — no blocking findings |

## Controller correction (gate verification)

The plan-adherence audit below states `tools/redis-key-guard.sh` FAILs on four
services. **That is a misread and is corrected here:** the guard was re-run from
the worktree root and exits **0 with zero violation lines**. The
`rediskeyguard: <path>` lines it emits are the tool's per-service *scan listing*
(it names each package it inspects), not failures. The Task 6 report's claim that
the guard passes is accurate. The full CLAUDE.md gate is green:
`go test -race`/`vet`/`build` clean, `docker buildx bake atlas-channel` passed
(bake was run even though no `go.mod` changed), and `redis-key-guard.sh` passes.

---

# Plan Audit — task-121-local-map-membership

**Plan Path:** docs/tasks/task-121-local-map-membership/plan.md
**Audit Date:** 2026-07-02
**Branch:** task-121-local-map-membership
**Base Branch:** main
**Implementation range:** 008396e97..e3903d817 (7 commits; plan committed at 008396e97)

## Executive Summary

All 6 plan tasks were faithfully implemented with file:line evidence for every
acceptance criterion. `go build`, `go vet`, and `go test -race` are clean across
the entire `atlas-channel` module, and every new test (session field providers,
map dedup/all-instances/transition) executes and passes on a forced (`-count=1`)
run. One extra commit beyond the plan's 6 (`90dd684aa`) is a legitimate,
in-scope test fixup for the map consumer whose atlas-maps REST mock was made
dead by Task 3. Recommendation: READY_TO_MERGE.

> Controller note: the reviewer's original redis-key-guard "FAIL" is superseded
> by the Controller correction above — the guard passes (exit 0).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Session field-filtering providers `InFieldModelProvider` / `InMapAllInstancesModelProvider` + tests | DONE | session/processor.go:66-79 (InFieldModelProvider, char!=0 && Field().Equals) and :81-94 (InMapAllInstancesModelProvider, world/channel/map match). Signatures match plan exactly. Tests: session/processor_test.go:748-895 (7 test funcs incl. instance discrimination, world/channel discrimination, characterless exclusion, cross-tenant, empty field, union). Commit e0c1429eb. |
| 2 | Login bootstrap `SetMapId`→`SetField`; `Processor.SetMapId` deleted; SetField tests replace SetMapId tests | DONE | kafka/consumer/session/consumer.go:190 `s = sp.SetField(s.SessionId(), f)`. `Processor.SetMapId` removed (grep finds only the internal Builder call at session/model.go:154 inside `setMapId`). Old `TestSetMapId`/`TestSetMapId_NonExistent` gone; `TestSetField_PreservesInstance` (session/processor_test.go:208) and `TestSetField_NonExistent` (:609) added. Commit 1b9fdbee5. |
| 3 | `map.Processor` recipient providers re-implemented over session providers; `characterIds` dedup helper; `requests` import removed; frozen signatures | DONE | map/processor.go:30-32 and :48-50 delegate to `p.sp.InFieldModelProvider` / `InMapAllInstancesModelProvider` via `characterIds(...)`. Dedup helper at :55-73 (seen-map). `requests` import gone. All 9 exported signatures unchanged; full module compiles. Tests: map/processor_test.go dedup/other/all-instances. Commit dd0609d46 (+ fallout 90dd684aa). |
| 4 | `TestTransition_WarpMovesRecipientSetAtomically` in map/processor_test.go | DONE | map/processor_test.go:136 — pre-warp {100,200} in A / empty B, `sp.SetField(bId, fB)`, post-warp A=[100] / B=[200]. Passes on forced run. Commit dd38ab23f. |
| 5 | Delete map/requests.go and map/rest.go; map package = processor.go + processor_test.go only | DONE | `git rm` both files (diff shows -29/-30 lines). `ls map/` returns only processor.go, processor_test.go. Greps for `requestCharactersInMap`, `_map.RestModel`, `_map.Extract`, `requests.` in map/ all return nothing. Commit 638c65495. |
| 6 | `field-transition-audit.md` with real verified line numbers, repo-relative paths, caller inventory | DONE | docs/tasks/task-121-local-map-membership/field-transition-audit.md present. All cited lines verified: Create at processor.go:311-313, consumer.go:190, character/consumer.go:249, model.go:152-156. Caller grep reproduced → 32 files (matches doc's "32 match, 31 actual callers excluding the definer"). Paths repo-relative; no absolute home paths. Commit e3903d817. |

**Completion Rate:** 6/6 tasks (100%) · **Skipped without approval:** 0 · **Partial implementations:** 0

### Note on commit count

The git range holds 7 commits vs. the plan's 6. The extra commit `90dd684aa`
("update map-consumer test for local session registry resolution (PS-2
fallout)") is required, in-scope fallout of Task 3: once
`fetchOtherCharactersInMap` sources local ids from the registry instead of the
atlas-maps REST mock, the two `kafka/consumer/map/consumer_test.go` tests'
`httptest` MAPS mock no longer intercepts anything. The fix registers sessions
in the local registry (world/channel 0 to match `session.NewSession`). Honest
completion of the change, not scope creep; the package still passes.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

---

# Backend Audit — atlas-channel (task-121 local-map-membership)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Scope:** changed Go files in range `008396e97..e3903d817`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-02
- **Build:** PASS · **Tests:** PASS (`go test -race ./map/... ./session/... ./kafka/consumer/map/...` all `ok`; vet clean)
- **Overall:** PASS

## Package Classification

The changed packages are socket-service infrastructure packages (`session`, `map`,
`kafka/consumer/*`). None is a classic DDD domain package: there is no `model.go`
+ `entity.go` + `rest.go` + `administrator.go` DB-backed resource here. `map/`
carries only a `Processor` over the in-process session registry; its former REST
adapter (`rest.go`, `requests.go`) was **deleted** by this change. Consequently the
DB/REST/administrator DOM items (DOM-01..05, 08, 10, 15..19) are **N/A**. The
items that do apply are checked below.

## Applicable Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `map/processor.go:21` and `session/processor.go:36` both `NewProcessor(l logrus.FieldLogger, ctx context.Context)`. |
| DOM-11 | Providers lazily evaluated | PASS | `session/processor.go:68-79` and `83-94` return `model.Provider[[]Model]` closures; the `GetInTenant` read is deferred until invoked. `map/processor.go:55-73` `characterIds` wraps the inner provider in another deferred closure. |
| DOM-13/14 | No cross-domain / direct-provider logic in handlers | PASS | Kafka consumer call sites call processor methods only; resolution logic lives in `map`/`session` processors. |
| DOM-20 | Tests present & structured | PASS (note) | Exhaustive per-case tests (exact-match incl. instance, world/channel discrimination, characterless exclusion, cross-tenant exclusion, all-instances union, dedup, atomic warp). Named test per behavior rather than a literal `[]struct{}` table. |
| DOM-21 | No duplication of atlas-constants types | PASS | New code uses shared `field.Model`, `world.Id`, `channel.Id`, `_map.Id` exclusively; the change **removes** a service-local reinvention (`map/rest.go` `RestModel`/`Extract`). |
| DOM-24 | Kafka producer stubbed in emitting tests | N/A | No direct or transitive emit in the changed test packages. |
| Multi-tenancy | Filter scoped to tenant, no cross-tenant leak | PASS | Both new providers read `getRegistry().GetInTenant(p.t.Id())`; proven by `TestInFieldModelProvider_ExcludesOtherTenant` (`session/processor_test.go:826`). |

## Task-Specific Invariant Verification

1. **No new goroutines/locks.** PASS. `Registry.GetInTenant` (`session/registry.go:76-89`) copies the tenant's sessions into a fresh `[]Model` under `RLock`/`defer RUnlock`; both providers filter that returned slice **outside** the lock.
2. **Mandatory character-id dedup.** PASS. `map/processor.go:55-73` `characterIds` deduplicates via a `seen map[uint32]struct{}`; proven by `TestCharacterIdsInMapModelProvider_DedupsCharacterIds`.
3. **Frozen exported signatures.** PASS. All ~40 external call sites across `kafka/consumer/*`, `skill/handler`, `door`, `merchant` compile unchanged (build + vet clean). Only the internal bodies were re-pointed at the session providers.
4. **Multi-tenancy correctness.** PASS. Scoped to `p.t.Id()`, cross-tenant exclusion test passes under `-race`.
5. **Shared atlas-constants types (DOM-21).** PASS.
6. **`SetMapId` removal is clean.** PASS. Public `Processor.SetMapId` deleted; its sole caller now calls `SetField(s.SessionId(), f)` (`session/processor.go:256-266`). The unexported `session/model.go:152 setMapId` remains, used only by `SetField`. Zero dangling references to deleted symbols.

## Field-Equality Correctness

`InFieldModelProvider` gates on `s.Field().Equals(f)` (compares
world+channel+map+**instance**), matching the deleted instance-scoped REST
endpoint. `InMapAllInstancesModelProvider` intentionally omits the instance
comparison (world+channel+map only), matching the deleted cross-instance
endpoint. Both additionally require `CharacterId() != 0`, excluding pre-login
sessions. Verified by `TestInFieldModelProvider_ExactMatchIncludingInstance`,
`TestInMapAllInstancesModelProvider_UnionsInstances`, and
`TestTransition_WarpMovesRecipientSetAtomically`.

## Summary

- **Blocking (must fix):** None.
- **Non-Blocking (informational):** the new provider tests use one named test per
  behavior rather than a literal `[]struct{}` table; `characterIds` hand-rolls its
  dedup loop (no dedup combinator exists, and `AllInChannelProvider` already uses
  the same hand-rolled filter idiom — consistent with existing convention).

**Overall: PASS.**
