# Backend Audit — packet-audit (task-081 off-by-one divergent remediation)

- **Scope:** Go changes in `tools/packet-audit/` for the "systematic off-by-one divergent remediation" lever
- **Review range:** `3afb59460` (BASE) → `95d8f886d` (HEAD)
- **Date:** 2026-06-10
- **Build:** PASS — `go build ./...` exit 0
- **Vet:** PASS — `go vet ./...` exit 0
- **Tests:** PASS — `go test -race ./...` all packages ok (cmd, idasrc, atlaspacket, csv, diff, report, template)
- **Overall:** PASS (one NON-BLOCKING robustness gap)

## DOM / service-shaped checks — N/A

`tools/packet-audit` is a developer CLI, not an Atlas microservice. There is no
DDD layering, GORM, JSON:API transport, Kafka, or multi-tenancy here. DOM-01..24,
SUB-*, EXT-*, SCAFFOLD-*, SEC-* are all **N/A** (not a service). This audit
evaluates Go correctness, string/JSON surgery safety, edge cases, and
determinism instead.

## Files reviewed

- `cmd/diff_shape.go` — `classifyDiff` + `diffShapeRun` (read-only diagnostic)
- `cmd/root.go` — `runDiffShape` flag wrapper
- `internal/idasrc/baseline_write.go` — `PrependCall` + `prependCallToCalls`
- `cmd/diff_shape_test.go`, `internal/idasrc/baseline_write_test.go`, `cmd/testdata/diffshape_mini.json`

---

## 1. `prependCallToCalls` JSON surgery (scrutinized hardest) — PASS for real inputs

`internal/idasrc/baseline_write.go:138-177`.

### Verified-safe properties (with reproduced evidence)

- **"calls" substring in a prior value/comment does NOT misfire.**
  `orderedFunctionRaws` (baseline_write.go:187-217) captures each object's bytes
  via `json.RawMessage`, so the bytes are still JSON-escaped. Any `calls` that
  appears inside a string value is necessarily written as `\"calls\":`
  (backslash-escaped quote), which does not contain the literal substring
  `"calls":` searched at line 139. I reproduced this with a value
  `x"calls": y` → serialized as `"x\"calls\": y"`; `prependCallToCalls` correctly
  anchored on the real key and produced valid JSON. A value `see calls: below`
  (no quote) likewise can't match because the search includes the leading quote.
  **PASS** — robust for any *valid* JSON baseline (which the exporter always
  produces).

- **Empty-array guard is correct.** Lines 148-152: `fb` = first `{`, `cb` = first
  `]`, both relative to `raw[open+1:]`; `fb < 0 || (cb >= 0 && cb < fb)` → "empty"
  error. Verified `[]` and `[\n   ]` (whitespace-only) both error out instead of
  corrupting. **PASS.**

- **Nested array/object inside the first element does NOT trip the empty guard.**
  For `[ {"op":..,"args":[1,2]} ]`, the first `{` precedes the first `]`
  (`[1,2]`'s close), so `cb < fb` is false. **PASS.**

- **Identical-body sibling collision avoided.** `PrependCall` (lines 113-132)
  mirrors `WriteDispatch`'s positional cursor walk (`strings.Index(text[cursor:], …)`,
  advancing `cursor` past each entry), so prepending to the *second* of two
  byte-identical entries does not corrupt the first. Same mechanism the existing
  B1 regression test guards for `WriteDispatch`.

- **Real-baseline round-trip is byte-safe.** I copied the committed
  `docs/packets/ida-exports/gms_v83.json`, ran `PrependCall` on 3 real entries,
  re-`json.Unmarshal`'d the whole file (valid), and re-parsed via the export
  source — each target gained exactly one leading call, non-targets unchanged.

- **The 54-handler data change is purely additive and valid.** `git show 491c7fd5c`
  is **+216 / −0** (exactly 4 lines × 54 prepends), zero removed lines, and all
  four baselines (`gms_v83/v87/v95/jms_185`) still parse as valid JSON. Each
  inserted block is well-formed `{"op":"Encode1","comment":"…"}`. This is strong
  evidence the surgery preserved every other byte.

### NON-BLOCKING robustness gap — `prependCallToCalls` corrupts non-canonical `[{` formatting

`internal/idasrc/baseline_write.go:155-156`.

`lineStart := strings.LastIndexByte(raw[:fb], '\n') + 1` then
`elemIndent := raw[lineStart:fb]` **assumes the first element's `{` is the first
non-whitespace token on its line**. If a calls array is written compactly with the
first element brace on the SAME line as the `[` — i.e. `"calls": [{` —
`elemIndent` captures the whole prefix `   "calls": [`, not just leading
whitespace, and the inserted block duplicates that text. I reproduced this:
input `"calls": [{\n  "op":…}]` produced **invalid JSON** containing a spurious
`"calls": [},` and a duplicated `"calls": [{`.

Why this is **non-blocking**:
- The exporter always emits `"calls": [\n    {\n` (bracket → newline → element on
  its own line); confirmed by grepping the real `gms_v83.json` / `gms_v95.json`
  (`'"calls": [\n    {\n     "op": …'`). The committed
  `TestPrependCall_SurgicalLeadingByte` exercises this canonical shape and passes,
  and the 54-handler live run produced valid JSON.
- So no current baseline triggers it.

Why it's still worth recording:
- `prependCallToCalls` guards the empty-array case but does NOT validate that
  `elemIndent` is whitespace-only. A hand-edited baseline (the comments elsewhere
  in this file explicitly anticipate hand-authored fields and mixed indentation)
  using the compact `[{` form would be silently corrupted into invalid JSON with
  no error returned.
- Cheap hardening: after computing `elemIndent`, return an error if
  `strings.TrimLeft(elemIndent, " \t") != ""` (i.e. the first `{` is not alone on
  its line), symmetric to the existing empty-array guard. Same applies to the
  `fb`-on-the-`[`-line assumption generally.

This is a latent fragility, not a live defect. No fix required to land the task;
flag for the file's robustness contract.

---

## 2. `classifyDiff` LCP/LCS math — PASS

`cmd/diff_shape.go:119-147`.

- **No index-out-of-range on any pathological input.** I exercised: both-empty,
  hand-empty/live-one, live-empty/hand-one, full-mismatch (same and differing
  lengths), all-identical-op grow/shrink, single-equal. None panicked. The suffix
  loop bound `s < n-p` (with `n = min(len(hand),len(live))`) guarantees the lowest
  index touched is `p ≥ 0` on the shorter side, so prefix and suffix never overlap
  and never index negative. **PASS.**
- **Classification is correct for the intended cases.** Committed
  `TestClassifyDiff` covers leading/trailing/interior/none with delta. Empty/
  full-mismatch inputs fall through to `interior` (sensible default), `none` only
  on exact equality via `eqOps`.
- **Known, acceptable ambiguity (not a bug):** for repeated identical ops
  (`hand=[D1,D1]`, `live=[D1,D1,D1]`) the greedy prefix consumes first, so the
  extra read is reported as `trailing` even though it is positionally ambiguous.
  The doc comment promises LCP+LCS, not leading-priority, and this is a read-only
  diagnostic that never changes a verdict, so it's acceptable. Worth knowing when
  reading reports: a leading off-by-one on an all-same-op list is labeled
  "trailing".

---

## 3. `diffShapeRun` read-only + deterministic — PASS

`cmd/diff_shape.go:25-93`.

- **Read-only w.r.t. the baseline / verdicts.** The only filesystem write is
  `os.WriteFile(opts.Report, …)` (line 87) — the report path, never the baseline.
  It calls `src.Entries()`, `ResolveLive`, `ExtractShape`, `ValidateShape`,
  `classifyDiff` — all read/pure. `verdict` is consumed only to *filter* rows
  (line 64), never written back. The committed
  `TestDiffShape_DeterministicAndReadOnly` asserts the baseline bytes are
  unchanged after two runs.
- **Deterministic output.** Addresses iterated in `sort.Strings(addrOrder)` order
  (line 42); final `rows` re-sorted by `fname` via `sort.Slice` (line 70), so the
  report ordering is independent of map iteration order. Same test asserts two
  runs produce byte-identical reports. **PASS.**
- Unresolvable bases are skipped (`continue`, line 58), matching validate's
  "unverifiable" treatment — no panic, no partial write.

## 4. `cmd/root.go` `runDiffShape` wrapper — PASS

`cmd/root.go:229-267`. Standard flag wrapper: requires `--version` and
`--report` (line 251), defaults baseline from version (line 256), builds the
real or instance-pinned MCP client, delegates to `diffShapeRun`. Mirrors the
sibling subcommand wrappers; no mutation, no surprises.

---

## Gate Results (run from `tools/packet-audit/`)

```
go build ./...      → exit 0
go vet ./...        → exit 0
go test -race ./... → ok (cmd, idasrc, atlaspacket, csv, diff, report, template)
```

(All scratch reproduction tests used during this audit were removed; working tree
is clean apart from this report.)

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- `prependCallToCalls` (`internal/idasrc/baseline_write.go:155-156`) corrupts
  baselines whose first calls element is on the same line as `[` (compact
  `"calls": [{` form), silently producing invalid JSON with no error. Not
  triggered by exporter-produced or any committed baseline, but a hand-edited
  compact baseline would break. Add a whitespace-only guard on `elemIndent`,
  symmetric to the existing empty-array guard.
