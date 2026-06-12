# Leaf Flat-Validation + Generalized Verbatim-Guard Dispatch — Design

**Task:** task-081-ida-export-reharvest (extension — recall lever #2)
**Date:** 2026-06-10
**Status:** Approved (brainstorming), pending implementation plan

## Goal

Reduce the **373 "per-mode shape not extractable"** unverifiable entries (49 base handlers)
that remain after per-branch verification, via two coupled sub-levers:

- **A — Leaf flat-validation:** a `#Mode` entry whose live function is a leaf (no multi-way
  dispatch) should be validated FLAT — its whole function IS its wire shape — instead of being
  short-circuited to `unverifiable`.
- **B — Generalized verbatim-guard dispatch:** extend the dispatch model from equality-only
  (`disc == N`) to match the **verbatim branch condition** of non-equality arms
  (inequality/range/flag), so those handlers can be inferred and verified.

## Background (established by scoping)

The 373 are `#Mode` baseline entries with no committed dispatch selector. By address occupancy:

- **~136 are SOLO** at their address (v83:21, v87:24, v95:63, jms:28).
- **~251 SHARE** an address with sibling `#Mode` entries (switch dispatchers).

The current `validate` rule (`cmd/validate.go`) marks *any* `#Mode` entry with empty dispatch
`unverifiable`:
```go
isMode := strings.Contains(e.FName, "#")
if isMode && (len(e.Dispatch) == 0 || len(live) == 0) {
    verdict = idasrc.ShapeUnverifiable
    detail = "per-mode shape not extractable (no usable dispatch selector)"
}
```
This is wrong for a solo `#Mode` entry whose function is a leaf: `ExtractShape(f, nil)` already
returns the whole function, which IS the wire shape, so `ValidateShape(hand, whole)` is the
correct flat comparison.

The shared-address entries are real switch dispatchers where `InferDispatchJoint` found no
selector — mostly **non-equality dispatch** (if/else on ranges/inequality, flag/bitfield,
indirect/vtable) the current `switch`+`if-==` model can't represent.

## The leaf-detection subtlety (the core correctness decision)

A naive "empty `CaseLabels` ⇒ leaf" is **unsafe**:
- A non-equality dispatcher (`if (x < 5) {…} else if (x < 10) {…}`) has empty `CaseLabels`
  today (the parser emits no equality cases for it). Flat-validating it would compare the whole
  function (all arms) against one mode's hand shape → **false divergence**.
- A lone optional-field `if` (`if (count) Decode4(...)`) must NOT count as dispatch — it's part
  of the single wire shape, so its function is still a leaf.

Therefore **leaf ≡ no multi-way dispatch**: the function has **no switch with ≥2 cases** and
**no if/else chain with ≥2 arms on one discriminator**. The parser computes this and exposes it
as `Fields.HasMultiwayDispatch`; `validate` uses it. This is why A and B are coupled — B's
parser extension (recognizing all dispatch arms) is what lets A detect dispatch reliably.

## Components

### 1. Parser — verbatim guards + multi-way-dispatch flag (`internal/idasrc/parse.go`)

Extend the if/else dispatch machine (from Task 2) so an arm with **any single-predicate
condition** emits the **verbatim condition text** as its guard:
- `if ( x < 5 )` → guard `x < 5`; `else if ( x >= 10 )` → `x >= 10`; `if ( x & 0x10 )` →
  `x & 0x10`; `if ( x != 3 )` → `x != 3`. Trailing bare `else` → `DefaultGuardToken`.
- Equality arms keep emitting `x == N` (unchanged — back-compat with existing selectors).
- **Bail to no-guard** only on conditions that are not a single readable predicate: compound
  (`&&`/`||`), indirect/`(*…)`, or non-discriminator expressions. Those arms stay honestly
  unverifiable.

Also expose **`Fields.HasMultiwayDispatch bool`**: true when the function has a `switch` with
≥2 `case` labels OR an if/else chain with ≥2 arms sharing one discriminator. Computed in the
same second pass as `collectCaseLabels` (a lone `if`/`if-else` with one dispatch arm is NOT
multi-way; a switch with a single case is NOT multi-way).

### 2. Selector — verbatim guard matching (`internal/idasrc/extract.go`)

Add `Guard string` to `Selector`:
```go
type Selector struct {
    Discriminator string `json:"discriminator,omitempty"`
    Case          int64  `json:"case"`
    Default       bool   `json:"default,omitempty"`
    Guard         string `json:"guard,omitempty"` // verbatim branch-condition clause
}
```
When `Guard` is set, `clauseMatches` matches a read whose composed guard contains that exact
clause (one of the `&&`-split, paren-trimmed clauses equals `Guard`). The
`Discriminator`/`Case`/`Default` paths are unchanged; a selector sets exactly one of
{`Case` (with `Discriminator`), `Default`, `Guard`}.

### 3. Inference — propose verbatim selectors (`internal/idasrc/infer.go`)

Generalize `enumerateCases` → `enumerateArms`: collect the distinct dispatch **guard clauses**
present across the base's reads (equality `disc == N` AND verbatim non-equality clauses AND the
default arm), not just numeric cases. `InferDispatchJoint` scores each hand shape against each
arm's `ExtractShape` and assigns one-to-one as today, but the resulting `Assignment.Dispatch`
carries a `{Guard: "<clause>"}` selector for a non-equality arm (or `{Discriminator, Case}` for
an equality arm, or `{Default: true}`). Confidence/joint logic unchanged.

### 4. Validate — leaf flat-validation (`cmd/validate.go`)

Replace the isMode short-circuit. For a `#Mode` entry `e` against resolved live `f`:
```
if e has a dispatch selector:
    live = ExtractShape(f, e.Dispatch)
    if len(live) == 0 -> unverifiable ("selector matched nothing")
    else -> ValidateShape(e.HandCalls, live)
else (empty dispatch):
    if !f.HasMultiwayDispatch -> ValidateShape(e.HandCalls, f.Calls)   // LEAF: flat-validate
    else -> unverifiable ("per-mode shape not extractable (no usable dispatch selector)")
```
Non-`#` entries are unaffected (already flat). The per-handler bijection accumulation is
unchanged (it only consumes entries that DO have a numeric selector).

### 5. resolve-dispatch / surgical writer — already pattern-agnostic

The surgical `WriteDispatch` round-trips arbitrary dispatch JSON, so a `{Guard: "x < 5"}`
selector persists with no change. `resolve-dispatch` auto-accepts high-confidence verbatim
selectors via the generalized inference (#3). No writer change needed.

## Bijection interaction (explicit limitation)

The case↔mode bijection (missing/extra-mode completeness) is **equality-based** — it diffs
numeric `CaseLabels` against bound numeric cases. A verbatim-guard arm has no numeric case, so
**completeness is NOT computed for non-equality handlers**; only per-arm correctness is. This is
an accepted limitation (correctness over completeness); the bijection code is unchanged and
simply never sees verbatim selectors as bindings.

## Plan opens with live characterization

Task 0 of the plan: sample the 251 shared-address handlers against the live IDBs
(ports 13337–13340) and classify each handler's dispatch — switch / if-equality /
if-non-equality-single-predicate / flag / indirect-vtable / nested — to confirm how much of the
251 Approach 1 covers and size the residual (indirect/vtable stays unverifiable). This replaces
guessing the parser scope with measured reality before the parser work.

## Testing

- **Parser:** verbatim-guard emission for `<`, `>`, `>=`, `<=`, `&`, `!=` arms + trailing
  `else`; `HasMultiwayDispatch` true/false cases (switch ≥2, if/else chain ≥2, lone optional
  `if` = false, single-case switch = false). Hand-crafted structural fixtures, consistent with
  the Task 2 convention; the real-Hex-Rays fixture hardening still owed from Task 2 applies here
  too and is exercised in the live characterization.
- **Selector:** `Guard` verbatim matching incl. composed `clause && loop n`.
- **Inference:** proposes a `{Guard}` selector for a non-equality arm; equality unchanged.
- **Validate:** solo-leaf (`!HasMultiwayDispatch`) flat-validates; solo-dispatcher
  (`HasMultiwayDispatch`, no selector) stays unverifiable; verbatim selector extracts and
  verifies.
- **E2E:** re-run validate on all four IDBs; measure the per-mode-not-extractable collapse.
- **Gates (CLAUDE.md):** `go test -race ./...`, `go vet ./...`, `go build ./...` on
  `tools/packet-audit`. Not a service → no docker bake; no redis.

## Out of scope (this pass)

- **Indirect/vtable dispatch** — no readable condition; stays honestly unverifiable.
- **Delegate-cycle / diamond descent (B2 from code review)** — separate focused follow-up.
- **Loop/opaque-block/mask divergent modeling (the 296 divergent)** — the other roadmap gap.
