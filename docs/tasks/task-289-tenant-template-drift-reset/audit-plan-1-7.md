# Plan Audit — task-289-tenant-template-drift-reset (Tasks 1-7)

**Plan Path:** docs/tasks/task-289-tenant-template-drift-reset/plan.md
**Audit Date:** 2026-09-02
**Branch:** task-289-tenant-template-drift-reset
**Base Branch:** main (merge-base 9613e7259)
**Scope:** Tasks 1-7 only (backend half: drift package, cross-package hash-equality guard, drift on the tenant read path, `ResetById`, and the REST reset endpoint). Tasks 8-14 are out of scope for this shard.

## Executive Summary

All 7 tasks in scope are fully implemented and match the plan's prescribed code almost verbatim, including exact comments and doc-strings. `go build ./...` and `go test ./...` pass cleanly for the entire `atlas-configurations` module (module root `services/atlas-configurations/atlas.com/configurations`), with every table-driven subtest named in the plan present and passing. The plan's checkboxes in `plan.md` are still all unchecked (`- [ ]`), which is a documentation-hygiene gap, not an implementation gap — git history (commits `fe0ecefea`..`cbf34d58e`) shows each task's step sequence (failing test → implementation → passing test → commit) was followed literally, one commit per task. The two deliberate rulings supplied in the audit brief (404-not-403 for cross-environment reset, and the FR-4.4 seven-field re-attachment) were verified against the code and both hold exactly as described.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Preset model parity (`ap`/`sp`) | DONE | `templates/characters/preset/rest.go` gained `AP uint16 \`json:"ap"\`` / `SP string \`json:"sp"\`` with the exact plan comment; `rest_test.go` new file with `TestAttributesCarriesAPAndSP`, passing. Commit `fe0ecefea`. |
| 2 | `drift` package — canonicalization/pruning | DONE | `drift/doc.go` and `drift/canonical.go` created verbatim to the plan's listing (`Doc`, `Excluded`, `Named`, `Properties`, `Canonicalize`, `prune`). `drift/canonical_test.go` covers all named subtests. Commit `4bc8c7e40`. `go test ./drift/...` passes. |
| 3 | `drift` hashing, comparison, merge, section validation | DONE | `drift/revision.go` (`Sections`, `Aggregate`, `Compare`, `All`) and `drift/merge.go` (`Merge`, `ValidateSections`, `ErrUnknownSection`) match the plan byte-for-byte. `templates/revision.go` diff is comment-only (verified via `git diff 9613e7259..0f1b3c75e -- templates/revision.go`, +7/-0, all inside a comment block) — the task-201 non-regression constraint holds. Commit `a944a51fd`. All `drift/revision_test.go` / `merge_test.go` subtests pass. |
| 4 | Cross-type equality between the two `RestModel`s | DONE | `drift/crosstype_test.go`, package `drift_test`, with `TestIdenticalDocumentsHashIdenticallyAcrossPackages`, `TestTenantOnlyDiagnosticsDoNotAffectHashes`, `TestWorldsDoNotAffectHashes` — all present and passing. Commit `e02af1411`. |
| 5 | `tenants.ViewRestModel` and baseline-decorated read paths | DONE | `tenants/rest.go:48-77` adds `ViewRestModel` with the five computed fields and the exact plan doc-comment. `tenants/processor.go` adds `ErrTenantNotFound`/`ErrNoBaselineTemplate` sentinels, `WithTemplates`, `baselineFor`, `decorate`, `makeView`, `ViewByIdProvider`, `AllViewProvider` (two-phase baseline resolution, per plan's NFR-1/FR-3.4 design). `tenants/view_test.go` — `TestView` (8 subtests) and `TestAllViewProviderResolvesEachBaselineOnce` (with the `countingTemplates` stub asserting `calls == 2`) both pass. Commit `1328efc34`. |
| 6 | `ResetById` | DONE | `tenants/reset.go` matches the plan's listing verbatim, including the discard-validator-mutation strategy, the FR-4.4 seven-field re-attachment (`Id`, `Environment`, `Region`, `MajorVersion`, `MinorVersion`, `Worlds`, `Diagnostics` — `reset.go:107-113`), history-before-write + outbox via `database.ExecuteTransaction`, and structured logging. `tenants/reset_test.go` implements all 11 plan subtests plus `TestResetById_PersistsBaselinePresetIdsVerbatim` and `TestResetById_ValidationFailureIsNotPersisted`; all pass. The plan's `CrossEnvironmentIs403` subtest was correctly rewritten to `CrossEnvironmentReadIs404` (`reset_test.go:353-378`), matching the pre-supplied ruling: `ResetById` reads via the environment-scoped `byIdEntityProvider`, so a cross-environment caller 404s before `update()`'s `AuthorizeWrite` — the only producer of `scope.ErrCrossEnvironmentWrite` in this call graph — is ever reached; the test explicitly asserts `errors.Is(err, ErrTenantNotFound)` and that `scope.ErrCrossEnvironmentWrite` is NOT surfaced. Commit `63664ef93`. |
| 7 | Reset route, handler, view-model read/create surfaces | DONE | `tenants/resource.go` registers `POST /{tenantId}/reset` (`resource.go:36`), adds `viewProcessor`, `writeJSONAPIError`, `resetRequest`/`parseResetSections`, and `handleResetConfigurationTenant` with the full sentinel-to-status switch (400 unknown section, 403 cross-env defense-in-depth, 404 not found, 409 no baseline, 422 validation-failure via `errors.As(*validationFailureError)`, else 500). GET list/GET-by-id/POST create switched to `ViewRestModel` (`resource.go:97,106,115,124,189,202`). `tenants/resource_reset_test.go` — `TestResetEndpoint` (12 subtests, including the plan's 11 plus the post-plan `ValidationFailureIs422` added in commit `ccc9656d9`), `TestGetTenantCarriesComputedAttributes`, `TestGetTenantsListCarriesComputedAttributes`, `TestCreateTenantReturnsViewModel`, `TestPatchIgnoresComputedAttributes` — all pass. Commit `cbf34d58e`. |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task in this shard's scope (1-7) was skipped, stubbed, or partially implemented.

One non-blocking documentation-hygiene observation: `plan.md`'s checkbox markers (`- [ ]`) for tasks 1-7 were never flipped to `- [x]` despite every step being completed and committed. This does not affect the code but means the plan file itself is out of sync with the actual state of the work; a reader of `plan.md` alone would (incorrectly) conclude nothing had been done.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-configurations (module root `services/atlas-configurations/atlas.com/configurations`) | PASS | PASS | `go build ./...` clean. `go test ./...` — all packages pass (`drift`, `templates`, `templates/characters/preset`, `tenants`, `tenants/characters`, `tenants/characters/preset`, `tenants/maplelife`, `tenants/socket`, plus unrelated packages `environmentcol`, `environments`, `outbox`, `scope`, `seeder`, `services`, `servicesuniq`, `socket`). No failures, no skips. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the corresponding audit of tasks 8-14, which is out of this shard's scope)

## Action Items

1. (Non-blocking, cosmetic) Update `plan.md` checkboxes for tasks 1-7 to `- [x]` to keep the plan document in sync with the completed, committed, and tested implementation.
