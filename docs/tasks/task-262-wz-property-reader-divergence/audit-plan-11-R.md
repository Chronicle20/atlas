# Plan Audit — task-262-wz-property-reader-divergence (Tasks 11-15 + R1/R2/R3)

**Plan Path:** docs/tasks/task-262-wz-property-reader-divergence/plan.md
**Audit Date:** 2026-08-26
**Branch:** task-262-wz-property-reader-divergence
**Base Branch:** main (range audited: a6820d1c4..HEAD)
**Scope:** Plan Tasks 11, 12, 13, 14, 15, plus added Tasks R1, R2, R3. Tasks 1-10 are out of scope for this shard.

## Executive Summary

The plan was re-scoped mid-execution (commit `0b3c8b21c`, "oracle withdrawal") after Task 5 found the supplied HaRepacker reference dump was never exported from the supplied `$WZ_ARCHIVE` — all 21 apparent divergences were `INPUT-MISMATCH`, not parser defects. Tasks 11, 12, 13, and 15 were withdrawn in place with internally coherent, evidence-backed reasons, and none of their forbidden deliverables (fixture rows for non-existent defects, the "decode fix," the `ErrPropertyOverrun` S1 sentinel) were landed — confirmed by grep returning zero hits for `ErrPropertyOverrun`/`ErrPropertyUnderrun` anywhere in `libs/atlas-wz`. Every requirement those tasks carried (FR-7, FR-9, FR-11, FR-12, FR-13, FR-14, and the S1/S2 strictness decisions) has an explicit, non-silent disposition in `prd.md` §4/§10, most substituted by Task R2's self-consistency gate. Task 14 (FR-8, sub-directory parse-failure propagation) was kept and fully implemented with matching tests. Tasks R1 (the re-scope itself), R2 (the whole-archive self-check), and R3 (post-review fixups) all landed exactly as their own review artifacts (`reviews/R1.md`, `R2.md`, `R3.md`) describe, and those reviews are already APPROVED with no blocking findings. Both affected modules (`libs/atlas-wz`, `services/atlas-data/atlas.com/data`) build clean and pass `go test ./... -count=1` and `go test -race ./...` in this shard's re-run. One minor documentation staleness was found: `prd.md` §10 still asserts "no `wztoxml` change was required," which became false once R3.2 (commit `36882bf08`) modified `adapter.go`'s log level — non-blocking, the change itself is a legitimate observability fix from Task 14's own review, not scope creep.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|-------------------|
| 11 | Failing regression fixtures (FR-9, FR-11, FR-12) | WITHDRAWN (coherent) | `plan.md:911-914` marks WITHDRAWN; reason: depended on `diagnosis.md` (Tasks 6-7, also withdrawn) naming byte patterns for defects that do not exist. `property_divergence_test.go` was never created — confirmed absent from the branch diff (`git diff --stat a6820d1c4..HEAD` lists no such file). No fixture rows for phantom defect classes were landed. |
| 12 | CHECKPOINT — the decode fix | WITHDRAWN (coherent) | `plan.md:982-984`. Reason: prerequisite Task 6 withdrawn, no `diagnosis.md` offset exists. Confirmed no speculative patch landed: `image.go`'s 165 inserted lines (checked below) are trace-hook/self-check instrumentation only, not a property-decode behavior change. |
| 13 | S1 strictness (`ErrPropertyOverrun`) | WITHDRAWN (coherent) | `plan.md:1016-1026`. Reason: Task 5 found 1136/1136 type-9 sub-objects clean in the 19 divergent images; Task R2 corroborated 15428/15428 clean across all 419 images in `$WZ_ARCHIVE` — no known trigger to justify a new production error surface. `grep -rn "ErrPropertyOverrun\|ErrPropertyUnderrun" libs/atlas-wz/` returns zero matches — confirmed not landed. |
| 14 | Propagate sub-directory parse failures; count ingest failures (FR-8) | DONE | `libs/atlas-wz/wz/directory.go:121-122` replaces the old `Warnf`-and-continue with `return nil, fmt.Errorf("parse sub-directory [%s]: %w", entryName, err)`. `libs/atlas-wz/wz/directory_error_test.go` has `TestSubDirectoryParseFailurePropagates` (line 111) and `TestValidNestedArchiveStillOpens` (line 142), both PASS when re-run. `services/atlas-data/.../wztoxml/adapter.go:29-46` adds per-image + per-archive failure counting (`serializeDirectory` returns `failed, total`); `adapter_test.go` has `TestSerializeDirectoryCountsFailures` and `TestSerializeDirectorySuccessLogsInfo`, both PASS. Blast-radius note in `plan.md:1130` names all `wz.Open`/`f.Root()` call sites; `wzcache.go:135` independently confirmed to check the error, remove the partial download, and touch no partial file. |
| 15 | Full-archive re-verification, S2 decision, final gate (FR-7, FR-13, FR-14) | WITHDRAWN (coherent) | `plan.md:1173-1181`. Reason: depends on Task 12's fix (withdrawn) and Task 5's allowlist (never produced — all divergences are `INPUT-MISMATCH`). S2 decision is moot for the same no-known-trigger reason as S1. No post-fix-diff.txt or S2 flip was fabricated — confirmed absent from the diff. |
| R1 | Re-scope task-262 after oracle withdrawal | DONE | Docs-only commit `0b3c8b21c` (six files under `docs/tasks/task-262-.../`, zero Go files — confirmed via `git show 0b3c8b21c --stat`). Independently reviewed and APPROVED in `reviews/R1.md` with all six binding constraints checked (repo-relative paths, line endings, no Go file touched, withdrawn requirements marked not deleted, Task 14 kept-vs-withdrawn judgment call assessed sound). |
| R2 | Whole-archive size-accounting self-check | DONE | Commit `8c86277c0`, six files (`wz/trace.go`, `wz/image.go`, `wz/trace_test.go`, `wzdiff/selfcheck.go`, `wzdiff/selfcheck_test.go`, `cmd/wzdiff/main.go`). `wzdiff.SelfCheck` (`libs/atlas-wz/wzdiff/selfcheck.go:54`) and `WriteSelfCheckReport` (`:103`) exist; `TestSelfCheckCleanArchivePasses` and `TestSelfCheckCorruptedArchiveFails` (`selfcheck_test.go`) both PASS on re-run. `reviews/R2.md` is APPROVED, no blocking findings, and independently re-ran the tool against the real archive (`images: 419 sub-objects: 15428 violations: 0 parse errors: 0`), matching the report. |
| R3 | Scope the `1136` figure, close two deferred minors | DONE | Two commits (`90994bd2a` docs-only R3.1/R3.3, `36882bf08` code R3.2). `reviews/R3.md` is APPROVED: verified `1136` is scoped to "19 divergent images" everywhere it appears in task docs, verified the `wzcache.go:135` blast-radius note addition matches an unchanged file, verified `adapter.go`'s summary line now logs `Infof` on full success / `Warnf` only when `failed > 0` (re-run independently below, PASS). |

**Completion Rate (this shard):** 8/8 tasks correctly disposed (1 DONE-implemented [14], 4 WITHDRAWN-coherent [11,12,13,15], 3 DONE-added [R1,R2,R3]) — 100% of in-scope tasks accounted for with no silent skips.
**Skipped without approval:** 0
**Partial implementations:** 0

## Requirement Disposition Check (FR-7/9/11/12/13/14, S1/S2)

All six FRs formerly carried by withdrawn Tasks 11/12/13/15 have explicit, non-silent dispositions in `prd.md`:

- **FR-7** (`prd.md:186-188`) — WITHDRAWN, general no-regression bar carried forward in Goals §2 and §10.
- **FR-9** (`prd.md:202-206`) — WITHDRAWN, general fixture-testing form carried forward to Task R2.
- **FR-11** (`prd.md:215-220`) — WITHDRAWN, Task R2 defines its own coverage (declared-size overrun/underrun on `wztest`-built archives).
- **FR-12** (`prd.md:221-224`) — WITHDRAWN, replaced by Task R2's own red-then-green TDD evidence.
- **FR-13** (`prd.md:228-231`) — WITHDRAWN, unreachable with these inputs even after a correct fix.
- **FR-14** (`prd.md:233-238`) — WITHDRAWN **as written only**; the `wzdiff` tool itself is explicitly kept and stated to remain useful for a future genuinely-matching dump.
- **S1/S2 strictness** — no dedicated FR number, but disposed in Task 13's/15's withdrawal text (`plan.md:1016-1026`, `1173-1181`) and corroborated by `prd.md §4.2` FR-8's restatement: the equivalent detection is delivered via Task R2's `wzdiff --selfcheck` as a read-only report rather than a new `Image.Properties()` production error path. `prd.md §10`'s live acceptance criteria (lines ~352-368) restate this as "zero size-accounting violations... reported by the gate," confirming the requirement's substance survived the withdrawal rather than vanishing.

No requirement was found to have vanished without disposition.

**One documentation staleness (non-blocking):** `prd.md:373` ("services/atlas-data builds and its tests pass; no wztoxml change was required") predates R3.2 (`36882bf08`), which did change `services/atlas-data/.../wztoxml/adapter.go` (log-level fix from unconditional `Warnf` to `Warnf`/`Infof` split). The change itself is legitimate and traces to a non-blocking finding in `reviews/task-14.md`, but the acceptance-criteria line was not updated afterward and is now factually incorrect. This is a doc-accuracy nit, not a scope or requirement-disposition problem — the underlying acceptance intent (atlas-data builds and tests pass) still holds and was verified.

## Skipped / Deferred Tasks

None. All withdrawn tasks (11, 12, 13, 15) carry explicit, evidence-cited withdrawal reasons written in place in `plan.md`, are internally coherent with `reference-fidelity.md`/`provenance.md`, and their forbidden deliverables were independently confirmed absent from the codebase (no `property_divergence_test.go`, no `ErrPropertyOverrun`/`ErrPropertyUnderrun`, no `post-fix-diff.txt`).

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-wz | PASS | PASS | `GOCACHE=/tmp/gocache-262 go build ./...` and `go test ./... -count=1` clean; `go test -race ./...` clean including `wz`, `wzdiff`, `wztest`. `TestSubDirectoryParseFailurePropagates`, `TestSelfCheckCleanArchivePasses`, `TestSelfCheckCorruptedArchiveFails` individually re-run and PASS. |
| services/atlas-data/atlas.com/data | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` clean across all packages. `TestSerializeDirectoryCountsFailures`, `TestSerializeDirectorySuccessLogsInfo` individually re-run and PASS. |

Note: `tools/verify.sh` was not run per dispatcher instruction (a gate is already running elsewhere); `execution-status.md` separately documents a pre-existing, branch-independent `golangci-lint`/go1.27 toolchain blocker on the flagless gate. Neither WZ archive was available in this environment to re-run the real-archive self-check numbers (`419`/`15428`/`0`/`0`); those figures were cross-checked against `reviews/R2.md`'s own independent re-run, which matches the task report verbatim.

## Overall Assessment

- **Plan Adherence:** FULL (for this shard's scope)
- **Recommendation:** READY_TO_MERGE (pending the other shard's Tasks 1-10 audit and the whole-branch `tools/verify.sh` gate, which is outside this shard's scope)

## Action Items

1. (Non-blocking) Update `prd.md`'s acceptance criterion "no `wztoxml` change was required" to reflect the R3.2 log-level fix, or strike it, so §10 stays internally accurate.
2. (Tracked already, not new) Rebuild `golangci-lint` against go1.27 so the flagless `tools/verify.sh` gate is reachable — `execution-status.md` already documents this as a pre-existing, branch-independent blocker.
