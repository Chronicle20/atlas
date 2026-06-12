# Task-081 — Validation pivot implementation plan (V1–V7)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development.
> Steps use checkbox (`- [ ]`) syntax. Companion: `design-validation-pivot.md` (read
> first — it locks the rationale, the empirical v83 measurement, and the schema).

**Goal:** Use the existing exporter (Phases 0/1/1.5) to **validate** the hand-authored
four-version baselines against the live IDB, per wire-shape (including `#`-mode
entries), producing a verification report of divergences — **never replacing**
hand-authored reads. A validator can only find problems, never regress the audit.

**Reuse:** the MCP-HTTP client (`mcphttp.go`), the direction-aware alias-tracked
parser (`parse.go`), descent + `Unresolved` (`harvest.go`), the `resolveWithVisited`
Delegate splicer + guard composition (`export.go`), and `diff.widthEquivalent`
(`internal/diff/diff.go`) — all unchanged. New surface: per-shape extraction, a
`dispatch` schema field, an auto-inference matcher, and a `validate` command.

**Key simplifier:** every hand-authored entry carries the base function's `address`,
and many `#`-entries share one address (all `OnFriendResult#*` → `0xa3f2e8`). The
validator decompiles **by address** (caching per address), sidestepping the
"synthetic name not found in IDB" problem entirely.

---

## Phase V-A — Offline buildable core (V1–V4, strict TDD, no live IDA)

> All V1–V4 work is unit-tested against committed fixtures
> (`internal/idasrc/testdata/real_onfriendresult_v83.c`, `real_sub_a40028_v83.c`) and
> fake clients. CI never touches a live IDB.

### Task V1: Per-shape extraction (`ExtractShape`)

**Files:**
- Create: `tools/packet-audit/internal/idasrc/extract.go`
- Create: `tools/packet-audit/internal/idasrc/extract_test.go`

- [ ] **Step 1: Write the failing test**

`ExtractShape(f Fields, dispatch []Selector) []FieldCall` returns the wire-shape reads
for one dispatch path: the **pre-branch** reads (empty guard, before the first
matched read — the discriminator + common header) followed by the **matched** reads
(guard satisfies every selector), in source order. Test against the real
`OnFriendResult` (resolve it via a fake client + `Harvest`, or `newExportSourceFromFile`
over a harvested `exportFile`):
```go
func TestExtractShapeOnFriendResultCase9(t *testing.T) {
    // Harvest OnFriendResult (+ sub_A40028; sub_4E4427 soft-fails -> Unresolved),
    // resolve it, then extract case 9.
    f := resolveRealOnFriendResult(t) // helper: Harvest+resolve the committed fixtures
    got := ExtractShape(f, []Selector{{Discriminator: "switch", Case: 9}})
    // pre-branch discriminator Decode1 (mode) + case-9 body, GW_Friend bulk = Unresolved:
    wantOps := []Primitive{Decode1, Decode4, DecodeStr, Unresolved, Decode1}
    if !opsEqual(got, wantOps) {
        t.Fatalf("extract case9 = %+v, want ops %v", got, wantOps)
    }
    // Other-case reads must be absent: no read carries a "switch == 0x14" guard.
    for _, c := range got {
        if strings.Contains(c.Guard, "== 0x14") || strings.Contains(c.Guard, "== 20") {
            t.Errorf("leaked a different case's read: %+v", c)
        }
    }
}
```

- [ ] **Step 2: Run → FAIL** (`ExtractShape`/`Selector` undefined).

- [ ] **Step 3: Implement**
```go
type Selector struct {
    Discriminator string `json:"discriminator,omitempty"` // "" matches any discriminator
    Case          int64  `json:"case"`
}

// ExtractShape returns the per-dispatch-path wire-shape reads from a resolved
// function: the pre-branch reads (empty guard, before the first matched read)
// plus the reads whose composed guard satisfies every selector. Empty dispatch
// returns all calls (a non-switch entry).
func ExtractShape(f Fields, dispatch []Selector) []FieldCall {
    if len(dispatch) == 0 {
        return append([]FieldCall(nil), f.Calls...)
    }
    firstMatch := -1
    for i, c := range f.Calls {
        if guardSatisfies(c.Guard, dispatch) { firstMatch = i; break }
    }
    var out []FieldCall
    if firstMatch >= 0 {
        for i := 0; i < firstMatch; i++ {
            if f.Calls[i].Guard == "" { out = append(out, f.Calls[i]) } // pre-branch common reads
        }
    }
    for _, c := range f.Calls {
        if guardSatisfies(c.Guard, dispatch) { out = append(out, c) }
    }
    return out
}
```
`guardSatisfies(guard, dispatch)`: split `guard` on `"&&"`; for EACH selector, require
some clause `X == V` where (`sel.Discriminator == ""` or `X == sel.Discriminator`) and
`parseIntLit(V) == sel.Case`. `parseIntLit` handles decimal and `0x` hex (so
`{Case: 9}` matches `"switch == 9"`, `{Case: 10}` matches `"switch == 0xA"`).

- [ ] **Step 4: Run → PASS**; full package `go test ./internal/idasrc/ -race`.
- [ ] **Step 5: Commit** `task-081(V1): per-shape extraction (ExtractShape + dispatch selectors)`.

### Task V2: `dispatch` schema field + resolver wiring

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/export.go` (add `Dispatch []Selector` to `exportFn`)
- Modify: `tools/packet-audit/internal/idasrc/export_test.go`

- [ ] **Step 1: Failing test** — load an export fixture whose entry has
  `"dispatch": [{"discriminator":"switch","case":9}]`; assert the parsed `exportFn`
  carries the selector and that a new `ResolveShape(fname)` (resolve + `ExtractShape`
  by the entry's own `Dispatch`) returns the case-9 shape.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** — add `Dispatch []Selector `json:"dispatch,omitempty"`` to
  `exportFn`; add `func (s *ExportSource) ResolveShape(ctx, fname) (Fields, error)`
  that resolves then `ExtractShape(resolved, entry.Dispatch)`. Leave `Resolve`
  unchanged (full function). Confirm round-trip JSON marshal preserves `dispatch`.
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** `task-081(V2): dispatch-selector schema field + ResolveShape`.

### Task V3: Auto-inference matcher (`InferDispatch`)

**Files:**
- Create: `tools/packet-audit/internal/idasrc/infer.go`
- Create: `tools/packet-audit/internal/idasrc/infer_test.go`

- [ ] **Step 1: Failing test** — given the resolved base `OnFriendResult` and a
  hand-authored read list resembling the `#Invite` reads
  (`[Decode1, Decode4, DecodeStr, ...]`), `InferDispatch` returns
  `dispatch=[{case:9}]` with high confidence; given an `#Update`-like single-read list
  it returns the matching case; an unmatchable list returns low confidence +
  candidate cases.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement**
```go
// InferDispatch finds the dispatch selector whose extracted shape best matches a
// hand-authored read list, using audit-grade equivalence (widthEquivalent;
// Unresolved acts as a wildcard segment). Returns the best selector path, a
// confidence in [0,1], and the close-runner-up cases when ambiguous.
func InferDispatch(base Fields, hand []FieldCall) (dispatch []Selector, confidence float64, candidates []int64)
```
Enumerate distinct case values across `base.Calls` guards (top-level discriminator).
For each, `ExtractShape(base, [{case}])` → score vs `hand` (sequence equivalence:
positions equal under `diff`-grade width tolerance; an `Unresolved` live read matches
any single hand read or a run — count as soft-match). Best score → selector;
confidence = best/(best+secondBest+epsilon); ambiguous if top two are within a small
margin. (Nested discriminators deferred to V-B against real nested shapes — single
level covers the v83 `#`-handlers.)
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** `task-081(V3): dispatch auto-inference matcher`.

### Task V4: `validate` command + per-shape diff + report

**Files:**
- Create: `tools/packet-audit/cmd/validate.go`
- Create: `tools/packet-audit/cmd/validate_test.go`
- Modify: `tools/packet-audit/cmd/root.go` (dispatch `args[0]=="validate"`)
- Create (maybe): `tools/packet-audit/internal/idasrc/shapediff.go` (+ test) for `ValidateShape`

- [ ] **Step 1: Failing test (fake client + tiny annotated baseline fixture)** —
  `cmd/testdata/validate_mini.json`: a base function + two `#`-entries with `dispatch`
  selectors + hand-authored `calls`; a `fakeMCP` (local, like `export_test.go`) serving
  the base decompile by ADDRESS. `validateRun(opts, client, w)` produces a report:
  one entry `verified` (extracted == hand under tolerance), one `divergent`
  (hand differs), and exercises an `unverifiable` (Unresolved/undecompilable) row.
  Assert the report counts + that NO hand-authored `calls` are mutated.
- [ ] **Step 2: Run → FAIL** (`validateRun` undefined).
- [ ] **Step 3: Implement**
  - `ValidateShape(hand, live []FieldCall) (ShapeVerdict, detail string)` reusing the
    `diff` width-equivalence logic: per-position compare with `widthEquivalent`;
    `Unresolved` live segment → `Unverifiable` for that span (not `Divergent`);
    length/op divergence → `Divergent` with the hand-vs-live detail; else `Verified`.
  - `validateRun(opts, client, stdout) int`: load the baseline export
    (`--baseline docs/packets/ida-exports/<version>.json`); group entries by
    `address`; for each address, `DecompileFunction(address)` ONCE (cache), parse with
    the entry's `Direction`, resolve; for each entry at that address, `ExtractShape`
    by its `Dispatch` (empty dispatch → whole function), `ValidateShape` vs the entry's
    `calls`; collect verdicts. Honor soft-fail: an undecompilable base → all its
    entries `unverifiable(decompilation failed)`; never abort. Deterministic report
    (sorted) to `--report <path>` + a stderr roll-up
    (`verified / divergent / unverifiable` counts).
  - `root.go`: add the `validate` subcommand flag set (`--version`, `--baseline`,
    `--ida-url`, `--ida-timeout`, `--report`), build the real `MCPHTTPClient`, call
    `validateRun`.
- [ ] **Step 4: Run → PASS**; whole module `go test ./... -race`, `vet`, `build`.
- [ ] **Step 5: Commit** `task-081(V4): validate command — per-shape extract + audit-grade diff + report`.

### Phase V-A GATE
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `tools/packet-audit`.
- [ ] V1 extraction proven on the real `OnFriendResult` case-9; V4 report proven on the
  annotated fixture (verified + divergent + unverifiable). No hand-authored `calls`
  mutated anywhere. Do not proceed to live phases until green.

---

## Phase V-B — Bootstrap selectors (LIVE IDA, maintainer-cycled)

> Requires the maintainer to load each IDB one at a time. Start with **v83 as a
> proof**; scale to the other three only after the v83 report is confirmed net-positive.

### Task V5: Auto-infer + commit dispatch annotations

**Files:**
- Modify: `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` (add
  `dispatch` to `#`-entries only; `calls` UNCHANGED)
- Create: `docs/tasks/task-081-ida-export-reharvest/selector-inference-<version>.md`
  (the proposed map + confidence + the human-confirmed ambiguous cases)

- [ ] **Step 1 (per version, v83 first):** add an `infer` mode (or a one-shot script
  using V3) that, per `#`-entry, decompiles the base by address, resolves, and
  `InferDispatch` against the entry's hand-authored `calls`. Emit a proposed selector
  map with confidence.
- [ ] **Step 2:** human-review the low-confidence / ambiguous rows (cite the case
  label + address); finalize.
- [ ] **Step 3:** write the finalized `dispatch` annotations back into the version's
  JSON (`#`-entries only). `calls` and all other fields stay byte-identical.
- [ ] **Step 4 (GATE per version):** re-load the JSON; confirm only `dispatch` keys
  were added (`git diff` shows additive `"dispatch": [...]` lines, no `calls` change).
- [ ] **Step 5: Commit** per version `task-081(V5): dispatch selector annotations (<version>)`.

**Phase V-B GATE:** every `#`-entry that is decomposable carries a `dispatch`
selector; ambiguous/undecomposable ones are documented (no silent guesses). `calls`
untouched.

---

## Phase V-C — Validate + triage (LIVE IDA)

### Task V6: Run validation per version; triage divergences

**Files:**
- Create: `docs/packets/validation/<version>.md` (the validation report)
- Modify (only on a real fix): `libs/atlas-packet/...` + byte tests; or a corrected
  baseline `calls` entry (hand-tracing error, with IDA evidence)

- [ ] **Step 1 (v83 proof first):**
```bash
packet-audit validate --version gms_v83 \
  --baseline docs/packets/ida-exports/gms_v83.json \
  --ida-url http://192.168.20.3:13337/mcp \
  --report docs/packets/validation/gms_v83.md
```
- [ ] **Step 2: Confirm net-positive.** The report must be `verified`-dominated with a
  bounded `divergent` list and honest `unverifiable` rows. If the report is dominated
  by spurious divergences, STOP — fix the extractor/inference (back to V-A) before
  scaling. (This is the gate the v83 measurement demands: validation must *find*
  real issues, not manufacture noise.)
- [ ] **Step 3: Triage every `divergent` row** (per the original plan's Phase-3/4
  discipline): hand-decompile in IDA; classify exactly one of:
  - **real Atlas bug** → fix `libs/atlas-packet/...` with a per-version byte test
    (`pt.Variants`/`pt.RoundTrip`, modeled on `socket/clientbound/hello_test.go`);
    re-validate → the row becomes `verified`.
  - **hand-tracing error** → correct that one baseline `calls` entry, citing the IDA
    address; re-validate → `verified`.
  - **representation-equivalent** → should already be tolerated by `ValidateShape`; if
    it surfaced, tighten the tolerance and re-run.
- [ ] **Step 4: Scale to v87/v95/jms185** (maintainer-cycled) only after the v83 report
  is clean and its divergences are dispositioned.
- [ ] **Step 5: Commit** per version `task-081(V6): validate <version> + triage`.

**Phase V-C GATE:** every `divergent` row dispositioned (fixed-atlas / fixed-baseline /
verified-equiv); changed `libs/atlas-packet` modules pass `go test -race`/`vet`/`build`
and `docker buildx bake` for each service whose `go.mod` changed; bugs too large for
this task registered as follow-ups (verify the number against `git log --all`).

---

## Phase V-D — Ledger + docs + verify

### Task V7: Document + verify
- [ ] **Step 1:** update `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` with the
  `validate` workflow, the `dispatch` selector schema + annotation procedure, the
  by-address decompile model, and the `export` (bootstrap-a-new-version) vs `validate`
  (verify-an-existing-baseline) distinction.
- [ ] **Step 2:** update both `_pending.md` registries: the `#`-mode entries are now
  *live-verified*, not hand-trusted; cite the validation report.
- [ ] **Step 3:** final verification — `go test -race ./...` / `vet` / `build` in every
  changed module; `docker buildx bake` for every service whose `go.mod` changed;
  `GOWORK=off tools/redis-key-guard.sh`.
- [ ] **Step 4:** code review — `superpowers:requesting-code-review`
  (`backend-guidelines-reviewer` + `plan-adherence-reviewer`); findings to `audit.md`.
- [ ] **Step 5: Commit** `task-081(V7): validation ledger + guide`.

**Phase V-D GATE:** all CLAUDE.md gates green; `#`-entries documented as live-verified;
code review run; `audit.md` present. Ready for PR.

---

## Self-review (coverage map)

- Extraction (per-case, no flatten) → V1; honored at resolve time → V2.
- Mode→case map without hand-annotating 118×4 → V3 (auto-infer) + V5 (bootstrap).
- Verify-not-replace; audit-grade tolerance → V4 (`ValidateShape`).
- Never overwrite hand-authored `calls` → V4 (report-only) + V5/V6 GATEs (additive diff).
- Real-bug → byte-test fix; hand-error → corrected baseline w/ IDA evidence → V6.
- v83-proof-before-scale → V6 Step 2 gate.
- Undecompilable/indirect → honest `unverifiable` (reuses Phase-1.5 soft-fail).
- Docs + the new-version-pass guide → V7.

## Open questions (resolve in V-B/V-C against real shapes)
- Serverbound multi-send selector form (op-byte vs guard-label vs send-index) — model
  in V2/V4 once a real `Send*` branch shape is in hand.
- Nested dispatch (mode → sub-mode) — `ExtractShape` already takes a selector PATH;
  validate against a real nested case in V-B.
- Auto-inference ties (two cases, identical read shape) — fall back to human +
  case-label/address hints (V5 Step 2).
- Composition with `per-mob`/`per-pet` dispatcher prefixes — already prepended by the
  resolver before `ExtractShape`; confirm with a per-mob `#`-entry.
