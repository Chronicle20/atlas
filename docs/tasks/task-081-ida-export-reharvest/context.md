# Task-081 Context — IDA Export Re-Harvest

Companion to `plan.md`. Read this first; it captures the decisions, file map, and
gotchas an engineer needs before touching code.

## What this task is

Build a **real automated exporter** for `tools/packet-audit` (today it is two
stubs), use it to re-export the four client read-order baselines (GMS v83/v87/v95
+ JMS v185), re-run the audit on the corrected input, then triage every verdict
flip against IDA ground truth and fix what is genuinely wrong. Source of truth:
`prd.md` + `design.md` in this folder.

## The two stubs being replaced

| Stub | File:line | Replaced by |
|---|---|---|
| `runExport` → prints "requires --ida-source mcp", exit 3 | `tools/packet-audit/cmd/export.go:8-12` | Phase 1, Task 1.10 |
| `ParseDecompile` → `errors.New("not yet implemented")` | `tools/packet-audit/internal/idasrc/mcp.go:52-54` | Phase 1, Tasks 1.2–1.5 |
| `MCPClient` interface (2 methods, no impl) | `tools/packet-audit/internal/idasrc/mcp.go:10-15` | Phase 1, Tasks 1.6–1.7 |

## Key plan-time design refinements (resolve design.md ambiguities)

The design (§3.2 vs §3.4) is ambiguous about whether the parser **inlines** helper
reads or **emits `Delegate` entries**. This plan resolves it:

1. **The exporter emits `Delegate`/`Ref` entries; the existing resolver inlines.**
   `resolveWithVisited` (`internal/idasrc/export.go:87-137`) is already built,
   tested, cycle-guarded, and guard-ANDing (task-065 item 8). We do **not**
   reimplement inlining. The producer that was "missing" (design §1 fact 1) emits
   `{op:"Delegate", ref:"<helperFName>"}` rawCalls and adds the helper to the
   export as its own keyed entry. Audit-time `ExportSource.Resolve` splices them.
   Benefit: structured, per-function-reviewable JSON; maximum reuse.

2. **`ParseDecompile` is pure and single-function.** It parses ONE function's
   Hex-Rays text and returns `[]rawCall` (the `export.go` type). It identifies the
   *packet variable* (the receiver/arg of `CInPacket::Decode*` calls). A call that
   **passes the packet variable** as an argument → emit `{op:"Delegate", ref:<callee>}`
   (a packet-reading helper). A call that does not → skipped. A known non-packet
   helper (denylist: `CUIFadeYesNo::*`, `StringPool::*`, `operator new`, alloc/UI)
   → skipped even if it spuriously passes the var. No MCP access inside
   `ParseDecompile` → trivially unit-testable from text fixtures.

3. **`Harvest` is the descent driver.** New function
   `idasrc.Harvest(ctx, client, roster, opts) (exportFile, error)`. BFS over the
   roster: for each FName → `GetFunctionByName` → `DecompileFunction` →
   `ParseDecompile` → `[]rawCall`; every `Delegate.Ref` discovered is enqueued as
   its own export entry; cycle-guard (visited set) + depth bound. `runExport`
   calls `Harvest`, then writes deterministic JSON. The four-version BuddyInvite
   regression (Task 1.9) tests `Harvest` against a fake client.

4. **`get_callees` / `StructInfo`** extend `MCPClient` (Task 1.6). `get_callees`
   resolves a callee name→address for recursion and confirms it is a real callee;
   `StructInfo` (`analyze_struct_detailed`) provides fixed-struct field layout for
   loop-vs-struct disambiguation when the helper body itself is opaque. Both are
   exercised via the in-package fake transport — never a live server in CI.

5. **`unresolved` is additive and honest (the anti-BuddyInvite invariant, §1.1).**
   New `idasrc.Primitive` value `Unresolved` (append to the enum so existing values
   keep their ordinals). New `diff.Verdict` value `VerdictUnresolved` ("🚫"). New
   `exportFn.Unresolved bool` JSON field. A function/element the parser cannot
   *prove* emits `Unresolved` — never a guess. The audit renders it as a distinct
   known-gap marker, not ✅/❌/⚠️/🔍.

## File map (Phase 1 — the buildable core)

| File | Action | Responsibility |
|---|---|---|
| `internal/idasrc/idasrc.go` | modify | add `Unresolved` Primitive (append to enum) |
| `internal/idasrc/export.go` | modify | add `exportFn.Unresolved` field; handle `op:"Unresolved"` in `resolveWithVisited` (emit `Unresolved` FieldCall, no error) |
| `internal/idasrc/parse.go` | create | `ParseDecompile(text) ([]rawCall, error)` + helpers (packet-var ID, read scan, loop/switch shape, denylist, unresolved fallback) |
| `internal/idasrc/parse_test.go` | create | fixture-driven unit tests |
| `internal/idasrc/testdata/*.c` | create | synthetic Hex-Rays fixtures (linear, count-loop, fixed-struct, mode-switch, cycle, non-packet-skip, unresolvable) |
| `internal/idasrc/mcp.go` | modify | extend `MCPClient` (`GetCallees`, `StructInfo` + result types); rewire `MCPSource.Resolve` for the `--verify-export` single-fn path |
| `internal/idasrc/harvest.go` | create | `Harvest(ctx, client, roster, opts) (exportFile, error)` descent driver (BFS, cycle/depth guard, unresolved accounting) |
| `internal/idasrc/harvest_test.go` | create | fake-client tests incl. four-version BuddyInvite regression |
| `internal/idasrc/mcphttp.go` | create | real `MCPClient` over MCP-HTTP (initialize → tools/call) |
| `internal/idasrc/mcphttp_test.go` | create | fake `http.RoundTripper` transport tests |
| `cmd/export.go` | rewrite | `runExport`: parse flags, build roster, call `Harvest`, write JSON, stderr unresolved summary |
| `cmd/root.go` | modify | export flags: `--ida-url`, `--ida-timeout`, `--version`, `--descent-depth`, `--output` |
| `cmd/export_test.go` | create | roster-build + determinism + flag tests (fake client) |
| `internal/diff/diff.go` | modify | `VerdictUnresolved` ("🚫"); a row whose IDA op is `Unresolved` (or fn `unresolved`) → `VerdictUnresolved` |
| `internal/diff/diff_test.go` | modify | unresolved-row verdict test |
| `internal/report/report.go` | modify (if needed) | ensure `Symbol()` renders 🚫; SUMMARY counts unresolved as a distinct bucket |

## Discovery-driven phases (2–7) — no fabricated code

Phases 2–7 act on artifacts that **do not exist until the live re-export runs**
(which packets flip, which bugs surface, which types decompose). The plan gives
each a **concrete per-item procedure + real templates + a hard gate**, not invented
test code. The byte-test template (Phase 4/5) is modeled on the real
`libs/atlas-packet/socket/clientbound/hello_test.go` `TestHelloWireShape` +
`pt.RoundTrip`/`pt.Variants` pattern.

## Hard prerequisite: Phase 0 rebase

This worktree forks from **pre-080** `main` (`40af0c80f`). task-080 is **not**
merged to `main`; the corrected baseline, A1–A5 analyzer work,
`STARTING_A_NEW_VERSION_PASS.md`, and the curated `_pending.md` live only on branch
`task-080-packet-audit-closeout` (local + origin). The `docs/packets/**` files
present in this worktree now are the **stale pre-080 versions**
(`STARTING_A_NEW_VERSION_PASS.md` is absent — proof). Phase 0 rebases onto
task-080 (or `main` once PR #678 merges) before any exporter work. **Do not start
Phase 1 until Phase 0's gate passes.**

## Key references (verified file:line)

- Export JSON schema + `resolveWithVisited`: `internal/idasrc/export.go:10-137`
- `rawCall` (op/comment/guard/ref): `internal/idasrc/export.go:38-49`
- `exportFn` (address/direction/dispatcher/calls): `internal/idasrc/export.go:10-36`
- `MCPClient` interface: `internal/idasrc/mcp.go:10-15`
- `ParseDecompile` stub: `internal/idasrc/mcp.go:49-54`
- `Primitive` enum: `internal/idasrc/idasrc.go:12-21`
- `FieldCall` / `Fields`: `internal/idasrc/idasrc.go:41-52`
- `candidatesFromFName` (roster source b): `cmd/run.go:190-366`
- audit pipeline `runPipeline`: `cmd/run.go:21-117`
- CLI dispatch (`export` subcommand): `cmd/root.go` (`Run`, first-arg check)
- `diff.Verdict` + `Symbol()`: `internal/diff/diff.go:10-21`; verdict logic `:33-75`
- byte-test oracle pattern: `libs/atlas-packet/socket/clientbound/hello_test.go`
- pt helpers: `libs/atlas-packet/test/roundtrip.go:12`, `libs/atlas-packet/test/context.go:18,26`
- loop-body guard convention (`"loop X"`): `internal/diff/diff.go:53`
- dispatcher prefixes (`per-mob`/`per-pet`): `internal/idasrc/export.go:172-191`

## Verify gates (CLAUDE.md)

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module.
- `docker buildx bake atlas-<svc>` from worktree root for every service whose `go.mod` changed (at least `tools/packet-audit`'s module if it has one; `libs/atlas-packet` consumers if wire fixes land; `atlas-configurations` if templates change).
- `tools/redis-key-guard.sh` clean from repo root (`GOWORK=off`).
- Backend nesting-cap clean.

## Project-memory gotchas that bite here

- **`reference_packet_audit_tool_mechanics`** — audit is FName × `candidatesFromFName` driven (templates are *dead code* for FName mapping); pass `--output docs/packets/audits` (the **parent**; tool appends `<region>_v<major>`); static diff is invalid for mask/mode packets → verify with byte tests.
- **`reference_ida_harvest_subagents`** — one IDB loaded at a time (user cycles); subagents can reach IDA-MCP; batch per-IDB; check JSON exports first.
- **`reference_rediskeyguard_invariant`** — run `tools/redis-key-guard.sh` with `GOWORK=off`.
- **`feedback_no_todos_in_deliverables`** — a too-large surfaced bug becomes a dedicated follow-up task, never a `// TODO` or a parked `_pending.md` "accepted" row.
- **`reference_task_numbers_historical_gap`** — if registering a follow-up, verify the next task number against `git log --all`, not just `task-numbers.sh next`.
