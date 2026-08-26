# Plan Audit — task-262-wz-property-reader-divergence (Tasks 1-10)

**Plan Path:** docs/tasks/task-262-wz-property-reader-divergence/plan.md
**Audit Date:** 2026-08-26
**Branch:** task-262-wz-property-reader-divergence
**Base Branch:** main (range audited: a6820d1c4..HEAD)
**Scope:** Tasks 1-10 only (Tasks 11-15 and the R-series are audited by a separate shard)

## Executive Summary

All 8 non-withdrawn tasks in this range (1, 2, 3, 4, 5, 8, 9, 10) are fully implemented with file-level evidence and passing tests. Tasks 6 and 7 are correctly marked `WITHDRAWN` in `plan.md`, and the withdrawal reasoning is internally coherent with `reference-fidelity.md` and `provenance.md`: Task 5 found all 21 disputed items to be `INPUT-MISMATCH` (the reference dump was never exported from `$WZ_ARCHIVE`), leaving no `PARSER-DEFECT` set for Task 6 to diagnose and no swallowed-directory bug for Task 7 to diagnose (9400300/9400301 are confirmed absent from the archive's own directory). `go build ./...`, `go test ./... -count=1`, and `go test -race ./...` all pass cleanly in `libs/atlas-wz`, and `services/atlas-data/atlas.com/data` builds and tests green. One minor documentation nit: every plan checkbox in this range is still `- [ ]` despite the corresponding work being committed and verified by other means (git log, code, tests).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Extract pure tree→XML mapping into `wz/wzxml` | DONE | `libs/atlas-wz/wz/wzxml/element.go:32,45,91` (`PropertiesToElements`, `PropertyToElement`, `FormatFloat`); `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:91-99` delegates via `wzxml.Element{...}` / `wzxml.PropertiesToElements(props)`; `runtime.go:101` call site unchanged; commit `c9b3b93f4`. |
| 2 | Nil-by-default trace hook on `*File` | DONE | `libs/atlas-wz/wz/trace.go:9-88` (`TraceEvent`, `SetTrace`, `traceHook`, nil-guarded emit helpers); `file.go:50-55` `trace` field with delegation-to-parent doc comment; `trace_test.go` has `TestSetTraceEmitsNodeEvents`, `TestTraceNilByDefault`, `TestSetTraceOnSubFileDelegatesToParent` (plus extra `TestSetTraceEmitsSubEventDeclaredActualEnd`, `TestTraceNilHookCostsNoExtraPosCalls`, `TestSetTraceEmitsUOLEvent`); commit `f780b23ad`, hardening in `4d0f73aa4`. `go test -race ./...` passes including `parse_race_test.go` invariants. |
| 3 | `wzdiff` core — tree model, XML loader, differ | DONE | `libs/atlas-wz/wzdiff/node.go:59` `FromElements`; `xmlload.go:18` `LoadImageXML`; `diff.go:21,33` `Delta.String()`, `Diff()`; `node_test.go`, `xmlload_test.go`, `diff_test.go` present; commit `e31b48f44`, review fixes in `c891ddc5e`. |
| 4 | `wzdiff` CLI and reference-resolution allowlist | DONE | `libs/atlas-wz/wzdiff/allowlist.go:41,126` `LoadAllowlist`/`Allowed`; `run.go:49,199` `Run`/`WriteReport`; `libs/atlas-wz/cmd/wzdiff/main.go` builds and `--help` lists `--archive`, `--reference`, `--allowlist`, `--trace` (plus an additional `--selfcheck` flag added later by the R-series, outside this shard's scope); commit `c09d6509a`, fixes in `2304a68dd`, `23d00d6eb`, `7fc7ab4f7`. |
| 5 | Reference-fidelity gate | DONE (outcome differs from original spec, as expected) | `docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md:381-408` — auditable 21-item table, all `INPUT-MISMATCH` (`PARSER-DEFECT` 0, `REFERENCE-RESOLUTION` 0, `MIXED` 0); `provenance.md` corroborates with independent Npc.wz/Reactor.wz count evidence. No `allowlist.tsv` was produced — correctly so, since there are zero `REFERENCE-RESOLUTION` deltas to allowlist. Commits `f7645c233`, `2c7405642`, `0b3c8b21c` (the re-scope commit). |
| 6 | Byte-level diagnosis of parser defects (FR-1, FR-2) | WITHDRAWN — coherent | `plan.md:568-572` withdrawal note is consistent with `reference-fidelity.md`'s 0 `PARSER-DEFECT` count; no `diagnosis.md` exists in the tree (`ls` confirms absence) and no defect-fix commits landed against `image.go`/`reader.go` in this range that would contradict "there are no parser defects to diagnose." |
| 7 | Diagnose the enumeration gap (FR-3) | WITHDRAWN — coherent | `plan.md:661-665` withdrawal note: `9400300.img`/`9400301.img` are absent from `$WZ_ARCHIVE`'s own directory (419 entries), confirmed at `reference-fidelity.md:404-405`. No enumeration-gap-specific diagnosis was produced or needed. (A later, unrelated commit `8e1f7201f` changes `directory.go:122` to propagate sub-directory parse errors instead of swallowing them — this is general robustness work, not a claim that the swallow caused the 419/421 gap, and does not contradict the withdrawal; it belongs to the R-series shard, not this range.) |
| 8 | `wztest.Builder` — scalar/vector/UOL/convex kinds | DONE | `libs/atlas-wz/wztest/builder.go:23-31` new `Kind*` constants, `:71-104` constructors (`Null`, `Short`, `Long`, `Float`, `Double`, `Vector`, `UOL`, `Convex`); `libs/atlas-wz/wz/wztest_kinds_test.go:17` `TestBuilderEmitsAllPropertyKinds`, `:105` `TestBuilderFloatZeroWithoutMarker` (plus extra `TestBuilderWzIntAndLongBoundaries`); commit `c40dd3b8f`. |
| 9 | `wztest.Builder` — dimensioned canvases with children | DONE | `libs/atlas-wz/wztest/builder.go:65` `CanvasWith(name, w, h, payload, children...)`; `libs/atlas-wz/wz/wztest_canvas_test.go:18` `TestBuilderCanvasWithDimensionsAndChildren`, `:98` `TestBuilderCanvasBackCompat`; commit `695422faa`. |
| 10 | `wztest.Builder` — offset-referenced string blocks | DONE | `libs/atlas-wz/wztest/builder.go:152` `SetStringDedup(on bool) *Builder`; `libs/atlas-wz/wz/wztest_dedup_test.go:136` `TestBuilderStringDedupRoundTrip`, `:155` `TestBuilderStringDedupOffByDefault`; commit `eec8247d8`, style fix `b7048596e`. |

**Completion Rate:** 8/8 applicable tasks (100%); 2/10 tasks correctly WITHDRAWN with coherent reasoning
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range are `SKIPPED` or `PARTIAL`. Tasks 6 and 7 are `WITHDRAWN`, and the withdrawal is substantiated:

- **Task 6** depended on Task 5 producing a non-empty `PARSER-DEFECT` set. Task 5's output (`reference-fidelity.md`) shows 0 `PARSER-DEFECT`/`MIXED`/`UNDETERMINED` labels across all 21 disputed items — every one is `INPUT-MISMATCH`, meaning the reference dump itself was not exported from the supplied archive. There is nothing for Task 6 to diagnose, and no `diagnosis.md` was fabricated to paper over the gap.
- **Task 7** depended on there being a genuine 419-vs-421 enumeration bug in `parseDirectory`. `reference-fidelity.md` records that `9400300`/`9400301` are simply not present in `$WZ_ARCHIVE`'s own directory (419 entries total) — not silently dropped. No enumeration-gap diagnosis was needed or produced.

Both withdrawals are traceable to the same root evidence (Task 5's byte-level adjudication plus the independent `provenance.md` cross-checks), and no task in this range's plan text or commits claims work that the withdrawal note says was never done.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| `libs/atlas-wz` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all packages `ok` (wz, wz/property, wz/wzxml, wzdiff, atlas, canvas, charparts, crypto, icons, manifest, mapimage, maplayout); `go test -race ./...` also clean. |
| `services/atlas-data/atlas.com/data` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all packages `ok`, including `atlas-data/data/wztoxml` and `atlas-data/reactor`. |

(`GOCACHE=/tmp/gocache-262` used per instructions to avoid the shared corrupted build cache.)

## Overall Assessment

- **Plan Adherence:** FULL (for Tasks 1-10 of this shard's scope)
- **Recommendation:** READY_TO_MERGE (pending the other shard's findings for Tasks 11-15/R-series, which this audit does not cover)

## Action Items

1. Cosmetic only: update `plan.md` checkboxes for Tasks 1-5, 8-10 from `- [ ]` to `- [x]` (or otherwise mark them complete) so the plan file reflects the verified state recorded in git history and this audit — no code or test changes required.
