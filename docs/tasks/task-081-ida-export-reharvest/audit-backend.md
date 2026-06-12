# Backend Audit — task-081 NET Go changes

- **Scope:** `git diff 3ab5d1dc5..c12f30985 -- '*.go'` (task-080 / task-083 are in the base, excluded)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-11
- **Build:** PASS
- **Tests:** PASS (all packages), `go vet` clean
- **Overall:** PASS

Two buckets:
1. `tools/packet-audit/**` — standalone cobra CLI / IDA-source parser. NOT an Atlas runtime microservice (no Kafka, no JSON:API REST, no GORM, no tenant context). Architecture-presuming DOM items are N/A.
2. `libs/atlas-packet/pet/serverbound/chat.go` — shared wire-encoding lib (merge-conflict resolution: pet-chat `updateTime` re-gated to GMS v95+).

## Build & Test Results

Evaluated against a detached worktree checked out at HEAD_SHA `c12f30985` (the live worktree is on `main`/`6da335a17`, a different lineage — the on-disk chat.go reverted to `MajorVersion() > 83`, which is NOT what this diff contains; all chat.go findings below were verified against the HEAD_SHA tree).

- `tools/packet-audit`: `GOWORK=off go build ./...` → exit 0. `GOWORK=off go test ./... -count=1` → all `ok` (cmd, atlaspacket, csv, diff, idasrc, report, template). `go vet ./...` → clean.
- `libs/atlas-packet`: `go build ./pet/...` → exit 0. `go test ./pet/serverbound/... -count=1` → ok.

## libs/atlas-packet/pet/serverbound/chat.go

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| GATE-01 | Version gate symmetric between Encode and Decode | PASS | Encode `chat.go:56` and Decode `chat.go:70` (HEAD_SHA) both gate `if t.IsRegion("GMS") && t.MajorAtLeast(95)` — identical predicate; write `WriteInt(updateTime)` / read `ReadUint32()` matched 4-byte widths. |
| GATE-02 | Gate value is the correct boundary | PASS | The same updateTime field on the sibling multi-chat packet gates identically at v95: `libs/atlas-packet/chat/serverbound/multi.go:54,71` (`t.MajorVersion() >= 95`). v95 is the established repo boundary for the DoAction updateTime int. |
| GATE-03 | Gate idiom matches repo convention | PASS (note) | HEAD_SHA uses `IsRegion(...)/MajorAtLeast(95)` (helpers at HEAD_SHA `libs/atlas-tenant/tenant.go:88,93`). The rest of `libs/atlas-packet` spells the equivalent predicate `t.Region()=="GMS" && t.MajorVersion() >= 95`. Semantically identical; helper form is a task-lineage convention, not a violation in the audited tree. |
| GATE-04 | Comment accurately documents the gate | PASS | `chat.go:54-55` (Encode) / `:68-69` (Decode): v95-only, CPet::DoAction 5 vs 4 encode calls, v84..v94 == v83, reconciles task-083 off-by-one. Consistent with the multi.go v95 sibling. |
| GATE-05 | Byte-level test backs the boundary | WARN | `chat_test.go TestChatRoundTrip` iterates `test.Variants` incl. GMS v83/v87/v95 (`test/context.go:18-24`) but uses `pt.RoundTrip` (`test/roundtrip.go:12`), a *symmetric* Encode→Decode check (asserts 0 unconsumed bytes). It does NOT assert absolute byte length/offset per version, so a both-sides-wrong gate would still round-trip green. Matches the package's universal round-trip convention; non-blocking. |

## tools/packet-audit/** (standalone CLI)

### N/A — architecture-presuming DOM checks (with reason)

| ID range | Reason N/A |
|----------|-----------|
| DOM-01..05 | No DDD model layer (builder/ToEntity/Make/Transform); it's a CLI. |
| DOM-06..09 | No HTTP handlers, no processor Impl pattern, no JSON:API input handlers. |
| DOM-10..11 | No GORM/DB/tenant; no providers. |
| DOM-12..17 | No HTTP/processor/administrator layering; cobra cmds write files (the tool's job). |
| DOM-18..19 | No JSON:API transport / REST models. |
| DOM-22 | Not a deployed service; no per-service Dockerfile. |
| DOM-23..24 | No Kafka producers/consumers anywhere. |
| SUB-*, EXT-*, SCAFFOLD-*, SEC-* | No sub-domains, no atlas-service HTTP clients, no new service scaffold, not auth-related. |

### Go-correctness / quality checks (applicable)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| GO-01 | Build clean | PASS | `GOWORK=off go build ./...` exit 0. |
| GO-02 | Tests pass | PASS | `GOWORK=off go test ./... -count=1` all ok. |
| GO-03 | `go vet` clean | PASS | `go vet ./...` no output, exit 0. |
| GO-04 | No swallowed `error` values | PASS | All 12 `_`-discards reviewed; none drop a meaningful `error`. `cmd/decompose.go:195` & `cmd/diff_shape.go:63` discard `ValidateShape`'s 2nd return — a *diagnostic string*, not error (`shapediff.go:45` → `(ShapeVerdict, string)`). `idasrc/conditional.go:77` discards the *2nd regexp* `pktVar`, not error (`parse.go:168` → `(prim, pktVar *regexp.Regexp)`). `mcphttp.go:202` best-effort `io.ReadAll` inside an already-failing error message; real path at `:206-208` checks err. `mcphttp.go:184` deferred Copy/Close cleanup. `baseline_write.go:173-174,236` & `cmd/resolve_dispatch.go:144` `json.Marshal` on internal always-marshalable structs. `cmd/run.go:1795` `_ = filepath.WalkDir` — inner callback never returns non-nil err; best-effort locator. |
| GO-05 | Error wrapping uses `%w` | PASS | `mcphttp.go:208,216`, `baseline_write.go:69`, etc. |
| GO-06 | No dead code after refactor | PASS | vet + compiler clean. `mcphttp.go:416 StructInfo` is a documented intentional interface-satisfying stub off the critical path (`_ = ctx/_ = name` silences unused-param), not dead code. |
| GO-07 | No TODO/FIXME/501 stubs (CLAUDE.md rule) | PASS | Zero `TODO`/`FIXME`/`panic(`/`not implemented` code stubs in changed non-test code (`export.go:43,109` "not implemented" are doc comments about packet feature absence). |
| GO-08 | Determinism claims hold | PASS | `idasrc/bijection.go:44-45` `sort.Slice` both output slices; inputs are slices. `baseline_write.go:45,108` map-ranges are validation-only (unknown-FName guard); the splice walks `entries` in *file order* via a positional cursor (`:59-76`). No nondeterministic map iteration reaches output. |
| GO-09 | Tests are real behavioral, not trivial | PASS | High assertion density (parse_test 70/18, mcphttp_test 39/15, diff_test 33/18, baseline_write_test 32/8, harvest_test 36/6). `infer_test.go:11-75` asserts confidence thresholds, ambiguity-candidate surfacing, Unresolved-wildcard absorption. `mcphttp_test.go` injects a real `http.RoundTripper` (`rtFunc` at `:14`) exercising request marshaling, session-id replay, handshake-once, 202 tolerance, error-envelope parsing — transport-level, not mocked out. |
| GO-10 | Idiomatic Go | PASS | Curried/pure helpers, no new global mutable state, cobra structure consistent with the existing tool. |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- **GATE-05 (WARN):** The pet-chat v95 boundary is covered only by a symmetric round-trip test, which cannot independently pin the absolute byte layout per version. A byte-level golden assertion (GMS v87 = no updateTime vs GMS v95 = +4 bytes) would catch a both-sides-wrong gate. Consistent with the package's existing round-trip-only convention, so not regressive.

### Informational (not a finding)
- The live worktree is on `main` (`6da335a17`); its on-disk `chat.go` uses `MajorVersion() > 83` (task-083 lineage), NOT this diff. All chat.go findings were verified against the audited HEAD_SHA `c12f30985`, where the gate is `MajorAtLeast(95)`.
