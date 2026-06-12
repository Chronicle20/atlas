# Backend Audit — packet-audit (leaf flat-validation + verbatim-guard dispatch lever)

- **Target:** `tools/packet-audit/` (developer CLI tool, NOT an Atlas microservice)
- **Range:** `8eba7578e68df47dc901df2b1d0bdc6607d4375e` .. `958457f897bd99d6dad0fe23c4bf7c5f4a526b4d`
- **Date:** 2026-06-10
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet ./...` exit 0)
- **Tests:** PASS (`go test -race ./...` exit 0, all packages ok)
- **Overall:** NEEDS-WORK (one false-verified correctness gap; non-blocking observations)

## Service-shaped DOM/SUB/EXT/SCAFFOLD/SEC checks — N/A

`tools/packet-audit` is a standalone developer CLI that audits MapleStory client
packet handlers against Atlas writers via IDA-MCP. It has no DDD layering, GORM
entities, JSON:API transport, Kafka, multi-tenancy, REST handlers, Dockerfile, or
k8s manifests. DOM-01..24, SUB-01..04, EXT-01..04, SCAFFOLD-01..08, SEC-01..04 are
all **N/A** — there is no service to apply them to. This audit covers Go
correctness, determinism, parsing/edge-case safety, and whether tests exercise
behavior, per the task framing.

## Gate Results (Phase 1)

```
go build ./...   -> exit 0
go vet ./...     -> exit 0
go test -race ./... -> exit 0 (cmd, atlaspacket, csv, diff, idasrc, report, template all ok)
```

No scratch/test artifacts leaked (`git status` clean for tools/packet-audit after
investigation).

## Findings

### FAIL-1 (Blocking) — Compound `else if` dispatcher is misclassified as a LEAF → false-verified risk

**Files:** `internal/idasrc/parse.go:818-824` (`collectCaseLabels` multiway
detection) and `internal/idasrc/parse.go:588` (`reIfCond` + `isSinglePredicate`
arm emission); consumed at `cmd/validate.go:132-140` (leaf flat-validate branch).

A genuine two-arm dispatcher whose **second arm uses a compound predicate**
(`else if (a && b)` — e.g. a range check `v5 > 10 && v5 < 20`, extremely common in
real Hex-Rays output) is classified `HasMultiwayDispatch = false`, because:

- `reElse` (`^\s*else\s*{?\s*$`) does not match `else if (...)` (trailing text).
- `reIfEq` does not match (`a && b` is not `ident == const`).
- `reIfCond` matches but `isSinglePredicate("a && b")` returns false
  (`parse.go:891` rejects `&&`), so the `else if` continuation branch at
  `parse.go:822` is never taken → `multiway` is not set.

Reproduced (scratch test, since removed):

```
if ( v5 < 5 )            -> Decode2  guard "v5 < 5"
else if ( v5 > 10 && v5 < 20 ) -> Decode4 guard ""   (arm guard DROPPED)
=> HasMultiwayDispatch = false
```

Two compounding problems:

1. The compound `else if` arm's read is emitted with an **empty guard**
   (`parse.go:588` bails to no `pendingArmFrag`), so the conditionally-read field
   leaks into `f.Calls` as if it were an unconditional pre-branch read.
2. Because `HasMultiwayDispatch == false`, an empty-dispatch `#Mode` entry over
   this function takes the leaf branch at `cmd/validate.go:132` and **flat-validates
   the whole function** via `ValidateShape(e.HandCalls, f.Calls)`.

`ValidateShape` (`internal/idasrc/shapediff.go:45-87`) is purely positional on
`.Op` and ignores guards entirely. So if the hand-authored `#Mode` reads happen to
equal the flat op union (`[Decode1, Decode2, Decode4]` in the repro), the entry is
reported **`verified`** when it should be **`unverifiable`** (a real multi-way
dispatcher with no usable selector).

This breaks the lever's stated safety invariant ("never a false `verified`"; see
`cmd/validate.go:118-121` and `parse.go:69`). Reachability is conjunctive (real
dispatcher + compound non-equality 2nd arm + empty dispatch on the `#Mode` entry +
hand shape coincidentally equals the flat union), so severity is **medium**, but it
is a true false-verified path and the entire design rests on that not happening.

**Suggested direction (not prescriptive):** treat a `reIfCond`/`reIfEq` header
carrying a leading `else` as multiway in `collectCaseLabels` *regardless* of
`isSinglePredicate` (the presence of `else if` already proves ≥2 arms — the
single-predicate gate is only relevant to whether a verbatim *guard* can be
emitted, not to whether the construct is multi-way). The existing `else if (==)`
path at `parse.go:820` already sets multiway unconditionally; the non-equality path
at `parse.go:822` should not be gated by `isSinglePredicate`.

### OBS-1 (Non-blocking) — `if/else` optional-field read flagged multiway (safe direction)

**File:** `internal/idasrc/parse.go:818`.

An `if (flag) { readA } else { readB }` that is a binary *optional-field* read (not
a mode dispatch) is flagged `HasMultiwayDispatch = true` (the bare `else` trips
`reElse` at `parse.go:818`). Verified via scratch test: such a function over an
empty-dispatch `#Mode` entry falls to `cmd/validate.go:141` → `unverifiable`. This
errs toward **unverifiable, never false-verified**, so it is the safe direction and
consistent with the documented conservative intent — recorded only for completeness.
There is no general way to distinguish a 2-way optional-field `if/else` from a 2-arm
dispatch without semantic analysis, so this is an accepted limitation, not a defect.

### PASS observations (evidence-cited)

- **reIfEq-before-reIfCond ordering holds** — equality is not shadowed by the
  verbatim fallback: `parse.go:576` checks `reIfEq` first, `else if` `reIfCond` at
  `parse.go:588`. Equality arms (`x == N`) always get the numeric guard, never a
  verbatim one. (`TestParseDecompile_IfElseDispatch`, `TestParseDecompile_NonEqVerbatimGuards` pass.)
- **Verbatim guard round-trips correctly** through emit → enumerate → extract:
  `enumerateArms` (`infer.go:304-328`) strips one paren pair + trims and stores
  `Selector{Guard: clause}`; `clauseMatches` (`extract.go:79-89`) applies the same
  strip+trim before an exact compare, and composed loop guards
  (`v5 < 5 && loop count`) still match the `v5 < 5` clause.
  (`TestExtractShape_VerbatimGuard`, `TestInferDispatchJoint_VerbatimArm` pass.)
- **No cross-matching** between verbatim and equality selectors: a `{Guard}`
  selector never matches an `==` guard and vice versa — `clauseMatches` returns
  early in the `sel.Guard != ""` branch (`extract.go:79-90`) and the equality loop
  `SplitN(clause,"==",2)` yields one part for a `<`/`&` clause (`extract.go:91-111`).
- **Bijection binding correctly skips Default and Guard selectors**
  (`cmd/validate.go:157-159`): `Case==0` verbatim/default selectors no longer
  pollute the equality case<->mode completeness check.
  (`TestValidate_VerbatimSelectorNotBijectionBinding` pass.)
- **Empty-leaf guard is correct** (`cmd/validate.go:133-137`): a leaf decompile
  yielding zero reads is `unverifiable` ("extraction failed"), not a false
  hand-N-vs-live-0 divergence. (`TestValidate_LeafEmptyLiveIsUnverifiable` pass.)
- **`enumerateArms` is deterministic** (first-seen order over `base.Calls`, keyed
  dedup map `infer.go:289-330`) and the `InferDispatchJoint` tie-break is
  deterministic (lower entry index, then lower column index `infer.go:164-168`).
  (`TestInferDispatchJointDeterministic` pass.)
- **Equality joint-resolution did NOT regress** — the canonical Invite-vs-Update
  8-vs-9 split still resolves with high confidence after `enumerateCases` ->
  `enumerateArms` swap. (`TestInferDispatchJointResolvesConflict`,
  `TestInferDispatchJointConfidenceJointAware`,
  `TestInferDispatchJointShortEntryStaysAmbiguous` pass.)
- **Switch multiway detection** is correct for the in-scope shapes: 2-case → true,
  1-case → false, case+default → true (`parse.go:874-879`).
  (`TestParseDecompileFields_HasMultiwayDispatch` pass.)

## Summary

### Blocking (must fix)
- **FAIL-1** — `parse.go:822` gates the non-equality `else if` multiway signal on
  `isSinglePredicate`, so a compound `else if` dispatcher reports
  `HasMultiwayDispatch=false`, drops the arm guard, and is flat-validated by
  `cmd/validate.go:132` → can be falsely `verified`. Breaks the "never false-verified"
  invariant.

### Non-Blocking (should fix / accept)
- **OBS-1** — optional-field `if/else` flagged multiway → `unverifiable` (safe
  direction; accepted limitation, no general fix without semantic analysis).

### Test coverage gap
- No test exercises a **compound `else if` dispatcher** (the FAIL-1 shape). The
  `TestParseDecompileFields_HasMultiwayDispatch` table (`parse_test.go:475-495`)
  covers only bare-`else`, `==`-chain, and switch forms. A fixture with
  `else if (a && b)` asserting `HasMultiwayDispatch=true` would have caught this and
  should be added alongside the fix.
