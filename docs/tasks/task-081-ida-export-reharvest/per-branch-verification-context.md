# Per-Branch Verification — Implementation Context

Orientation for the engineer executing `per-branch-verification-plan.md`. All paths are
relative to the worktree root `.worktrees/task-081-ida-export-reharvest/`.

## Where things live (tools/packet-audit)

| Concern | File | Key symbols |
|---|---|---|
| Core types | `internal/idasrc/idasrc.go` | `Direction`, `Primitive` (`Decode1..DecodeBuf`, `Unresolved`), `FieldCall{Op,Comment,Guard}`, `Fields{Function,Address,Direction,Calls}` |
| Per-branch extraction | `internal/idasrc/extract.go` | `Selector{Discriminator,Case}`, `ExtractShape(f,dispatch)`, `guardSatisfies`, `clauseMatches`, `parseIntLit` |
| Decompile → reads | `internal/idasrc/parse.go` | `ParseDecompile(text,dir)`; switch state machine: `swEntry{bodyDepth,discrim,caseIdx}`, `pendingSwitchVar`, `reSwitch`, `reCase`, `reBareVar`, scope stack, `composeGuard` |
| Inference | `internal/idasrc/infer.go` | `enumerateCases(base)→(disc,cases)`, `InferDispatch`, `EntryShape{FName,Hand}`, `Assignment{FName,Dispatch,Confidence,Candidates}`, `InferDispatchJoint(base,entries)`, `seqScore`, `FieldEquivalent` |
| Shape compare | `internal/idasrc/shapediff.go` | `ShapeVerdict` (`ShapeVerified/ShapeDivergent/ShapeUnverifiable`), `ValidateShape(hand,live)` |
| Baseline load | `internal/idasrc/export.go` | `exportFile{Binary,MD5,GeneratedAt,Functions map[string]exportFn}`, `exportFn{Address,Direction,Dispatcher,Notes,Unresolved,Absent,Dispatch []Selector,Calls}`, `NewExportSource`, `(*ExportSource).Entries()→[]BaselineEntry{FName,Address,Direction,Dispatch,HandCalls}`, `ResolveShape` |
| Live decompile | `internal/idasrc/live.go` | `ResolveLive(ctx,client,addr,dir,HarvestOpts{DescentDepth})→(Fields,error)` |
| Subcommand dispatch | `cmd/root.go` | `Run(args,stderr)`: string-prefix dispatch on `args[0]` (`export/validate/infer/decompose/triage`). Each `runXxx` builds a `flag.FlagSet`, an `idasrc.MCPClient` (`--ida-url` default `http://192.168.20.3:13337/mcp`, `--ida-port` for `select_instance`), delegates to a core `xxxRun`. |
| validate driver | `cmd/validate.go` | `validateOpts{Baseline,Report,DescentDepth}`, `shapeResult{FName,Verdict,Detail}`, `validateRun`, `writeReport`, `writeSection`. Mode rule: `isMode && (len(Dispatch)==0 || len(live)==0) → ShapeUnverifiable "per-mode shape not extractable"`. |
| infer driver | `cmd/infer.go` | `inferOpts{Baseline,Out,MinConfidence,DescentDepth}`, `inferProposal`, `writeProposals` (JSON `{proposals: {FName: {dispatch,confidence,candidates}}}`) |
| Fixtures | `internal/idasrc/testdata/*.c` (real decompile), `*.json` (mini baselines) | loaded via `mustFixture(t,name)` in `parse_test.go` |

## Facts that shape the plan

- **The `dispatch` JSON field already round-trips on load** (`exportFn.Dispatch`,
  `BaselineEntry.Dispatch`). What's missing is a *writer* that sets it on the committed
  baseline, and the *values* (no `#Mode` entry has one today).
- **Parser emits guards for `switch`/`case` (`disc == N`) and `for/while` loops (`loop N`)
  only — NOT `if/else`.** Confirmed in `parse.go`. This is why if/else-dispatched
  handlers (`OnCheckPasswordResult#*`) are unverifiable even though their selector grammar
  (`==`) is identical.
- **`enumerateCases` derives cases from guards on *reads*** — a dispatch case that reads
  nothing is invisible. Bijection needs the full case-label set from the *structure*.
- **`Selector` has no default/else representation.** A trailing `else` / `switch default:`
  arm cannot be selected today.
- **Verdict enum (audit JSON, distinct from `ShapeVerdict`):** `0=✅ 1=⚠️ 2=❌ 3=🔍 4=🚫`
  (`internal/diff/diff.go`). Not used by `validate`, which uses `ShapeVerdict`.

## Live IDB ports (for end-to-end)

| Version | `--ida-port` | baseline | audit dir |
|---|---|---|---|
| gms_v83 | 13337 | `docs/packets/ida-exports/gms_v83.json` | `docs/packets/audits/gms_v83` |
| gms_v87 | 13338 | `docs/packets/ida-exports/gms_v87.json` | `docs/packets/audits/gms_v87` |
| gms_v95 | 13339 | `docs/packets/ida-exports/gms_v95.json` | `docs/packets/audits/gms_v95` |
| gms_jms_185 | 13340 | `docs/packets/ida-exports/gms_jms_185.json` | **`docs/packets/audits/jms_v185`** (name mismatch — pass `--audit-dir` explicitly) |

MCP host reachable (HTTP 405 on GET = wants POST). One IDB per port; `select_instance`
multiplexes. Only one IDB is interactively loaded at a time historically — for harvesting
real fixtures, decompile through the live client and save the text to `testdata/*.c`.

## Verification gates (CLAUDE.md)

From `tools/packet-audit/`: `go test -race ./...`, `go vet ./...`, `go build ./...`.
**No `docker buildx bake`** (packet-audit is a tool, not a service — not in
`.github/config/services.json`). **No redis** → redis-key-guard N/A. Determinism matters:
reports/proposals must be byte-stable for identical (opts, client).

## Baseline snapshot (2026-06-09, the bar to beat)

validate, all four versions: **verified 293 / divergent 296 / unverifiable 508** (1097 total).
Of the 508 unverifiable, **450 are "per-mode shape not extractable"** — the target of this
plan. Success = that bucket collapses toward 0 and the new missing-mode / extra-mode buckets
populate with real, allowlist-filtered findings.
