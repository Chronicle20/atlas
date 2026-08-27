# Review: Task 10 fix round 1 (commit c5b76c6b9)

Range reviewed: `b1d090265..c5b76c6b9` (single commit `c5b76c6b9`). `b1d090265` was reviewed
separately (`review-task-10.md`) and is not re-reviewed here.

Inputs: `.superpowers/sdd/plan/task-10-brief-fix1.md`, `docs/tasks/task-263-backend-guideline-conformance/review-task-10.md`,
`.superpowers/sdd/plan/task-10-report.md` ("Fix round 1" section).

## Diff summary

```
docs/tasks/task-263-backend-guideline-conformance/codemod/transform.go       | 20 +++++++++++++++----
docs/tasks/task-263-backend-guideline-conformance/codemod/transform_test.go | 114 ++++++++++++++++++++
```

`transform.go` changes: (1) `typeChecksWithOverlay`'s `packages.Config` gains `Tests: true`
(`transform.go:543`); (2) `mergeIntoRestGo` restores CRLF when `original` contained it
(`transform.go:449-458`); (3) `mergeIntoRestTestGo` restores CRLF when `existing` contained it
(`transform.go:513-520`). `transform_test.go` adds: a `TestGenerateTransform` case with an
ill-typed generated round-trip test, a `TestGenerateRoundTripTest` case with same-typed sibling
fields, and two direct unit tests for the CRLF-preservation merge functions.

## 1. BLOCKING finding — is it actually closed?

**Yes, independently confirmed closed.**

The fix sets `Tests: true` on the `packages.Config` passed to `packages.Load` in
`typeChecksWithOverlay` (`transform.go:543`), which is byte-for-byte the same fix `rename.go`'s
`loadModuleForRename` already applies (`rename.go:285`) for the identical reason — reaching a
`_test.go`-only call site. The overlay itself was already correctly keyed at `restTestGoPath`
(`transform.go:137`) before this fix; the defect was purely that `packages.Load` never loaded the
test-augmented package variant to apply that overlay entry against, so the brief's specific
concern — "a `Tests: true` that is set but whose test-file overlay still does not reach the type
checker would be the same defect wearing a fix" — does not apply here: the overlay map and its
consumption path are unchanged by this commit, only the `Load` variant selection changed, and the
loop at `transform.go:546-550` already iterates all packages returned by `Load` (which, with
`Tests: true`, includes the test-augmented variant) and returns the first error found in any of
them — no variant-selection logic needed adding or fixing.

I did not take the report's word for it. I reproduced both sides myself, in this worktree, without
touching git history:

- **GREEN (current code):**
  ```
  $ cd docs/tasks/task-263-backend-guideline-conformance/codemod
  $ GOWORK=off go test ./... -run 'TestGenerateTransform/generated_round-trip_test_does_not_type-check' -v
  --- PASS: TestGenerateTransform (0.64s)
      --- PASS: TestGenerateTransform/generated_round-trip_test_does_not_type-check (0.64s)
  ```
- **RED (fix reverted in the working tree only, via `sed` removing the `Tests:   true,` line,
  restored with `git checkout -- transform.go` immediately after, confirmed clean with
  `git status --short`):**
  ```
  transform_test.go:249: GenerateTransform() err = <nil>, want a *skipError containing "does not type-check"
  --- FAIL: TestGenerateTransform (0.03s)
      --- FAIL: TestGenerateTransform/generated_round-trip_test_does_not_type-check (0.03s)
  ```

This matches the implementer's report exactly and matches the shape of the original reviewer's
repro (an unambiguous type error reachable only through the generated test body). Full module
`go build`, `go vet`, and `go test ./...` were also re-run independently and are green (all 8
`TestGenerateTransform` cases, all 4 `TestGenerateRoundTripTest` cases, both new CRLF unit tests
pass — `atlas-task-263-codemod  7.369s`).

## 2. Non-blocking — same-typed-sibling fixture

`transform_test.go:397-428` adds a fixture with two `uint32` fields (`x`, `y`) and two `string`
fields (`s1`, `s2`) and asserts four distinct, non-zero rendered literals: `x: 11,`, `y: 22,`,
`s1: "field3",`, `s2: "field4",`. This is not merely "generation succeeds" — each expected
substring is asserted individually via `strings.Contains` (`transform_test.go:439-443`), and the
four values are pairwise distinct within each type. I traced `deriveTestValues`
(`transform.go:267-313`): `n := i + 1` is a single index over the *global* mapping-order loop (not
reset per type), so `x`→n=1→`11`, `y`→n=2→`22`, `s1`→n=3→`"field3"`, `s2`→n=4→`"field4"` — this
matches the fixture's expectations exactly, and a regression that reset the index per type (which
is precisely the risk the prior review flagged) would produce `x:11,y:11` or `s1:"field1",
s2:"field1"` and fail this test. Closed.

## 3. Non-blocking — CRLF preservation

`mergeIntoRestGo` (`transform.go:449-458`) and `mergeIntoRestTestGo` (`transform.go:513-520`) both
now check `strings.Contains(original/existing, "\r\n")` and, if true, `bytes.ReplaceAll` all `\n`
to `\r\n` on the `format.Source`/`format.Node` output. This is the identical pattern to
`rename.go:253-255` (`bytes.Contains(orig, []byte("\r\n"))` / `bytes.ReplaceAll`), just applied to
an in-memory string return instead of a direct `os.WriteFile`. The two new unit tests
(`TestMergeIntoRestGoPreservesCRLF`, `TestMergeIntoRestTestGoPreservesCRLF`,
`transform_test.go:453-480`) each assert both that the output contains `\r\n` and that
`strings.Count(got, "\r\n") == strings.Count(got, "\n")` — i.e. no bare LF survives alongside the
restored CRLFs, which is a stronger and correctly-targeted assertion (a weaker "contains \r\n"-only
check would pass even with a mix of line endings). Ran independently, both pass. Closed.

## Dry-run re-run and skip legitimacy

I re-ran Task 10 Step 5's dry run myself, independently of the report, against this worktree:

```
GOWORK=off go run . transform -repo <worktree-abs> \
  -classification <worktree-abs>/docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  -dry-run -ledger /tmp/task10-fix1-verify-ledger.tsv
```

Result: 94 rows total, **76 APPLIED / 18 SKIPPED** — matching the report's tally exactly (skip
histogram cross-checked against the report's table: 7 "does not type-check", 6 "unsupported field
type", 3 "two or more bool fields", 1 "Extract maps no fields", 1 "Transform already declared" —
totals to 18). `git status --short services` after the run is empty; no real file was touched.

I spot-checked two of the six newly-skipped packages by reading the actual service source (not
the codemod's error message alone):

- **`services/atlas-cashshop/atlas.com/cashshop/character`** (byte-range overflow). `rest.go:88-120`'s
  `Extract` maps 29 fields in order; the 29th is `stance` (`Stance byte` at `rest.go:40`).
  `deriveTestValues` (`transform.go:302-303`) assigns integer fields `n*11` where `n` is the
  1-based mapping-order index, so `stance` gets `29*11 = 319`, which overflows a `byte` (max 255).
  This is exactly what the reported error says (`rest_test.go:38:23: cannot use 319 ... overflows`)
  and is a real defect the old ungated code would have silently shipped as an untyped-constant
  overflow error at compile time on the real tree — a genuine catch, not a false skip.
- **`services/atlas-maps/atlas.com/maps/data/map/info`** (builder `Build()` return-arity mismatch).
  `model.go:29-36` defines `func (b *Builder) Build() Model` — a single return value, no error.
  `buildRoundTripTestFunc` (`transform.go:401-406`) unconditionally renders
  `m, err := NewBuilder()....Build()` whenever `useBuilder` is true, assuming `Build` always
  returns `(Model, error)`. `builderCoversAllFields` (`transform.go:344-361`) confirms a `Build`
  method exists via `types.LookupFieldOrMethod` but never inspects its signature's result count or
  types — so this package would previously have been accepted with `useBuilder = true` and shipped
  a `rest_test.go` with an assignment-mismatch compile error. Also a genuine catch.

Both spot-checked packages would genuinely have produced uncompilable generated tests under the
old (pre-fix-round-1) gate; neither is a false skip from an over-broad check. I did not
independently re-derive the other four ("does not type-check") skips or the 11 "unsupported field
type"/"two or more bool fields"/"Extract maps no fields"/"Transform already declared" skips — those
gates are unchanged by this commit and were already reviewed and approved in `review-task-10.md`.

## Not evaluable

- The other four newly-skipped-by-this-fix packages (`atlas-channel/character`,
  `atlas-messages/character`, `atlas-monsters/monster/consumable`, `atlas-npc-shops/character`)
  were not independently traced by hand; only two of the six were spot-checked per the brief's
  "at least two" instruction. Their error messages are structurally identical to one of the two
  confirmed root causes (byte overflow for `atlas-channel/character`'s `330`; `Build` return-arity
  for the other three), which is consistent but not independently re-derived from source for all
  six.
- Whether any of the 76 still-APPLIED packages contain a *different*, not-yet-discovered class of
  generated-test defect that this gate does not catch is out of scope for this fix-round
  confirmation review (unchanged surface from `review-task-10.md`, already covered there under
  "Verified as correct").

## Verdict

APPROVED. All three items from `task-10-brief-fix1.md` are closed: the blocking type-check gate
now genuinely reaches the generated test file (independently reproduced RED and GREEN), the
same-typed-sibling fixture asserts four distinct non-zero values via per-field-index tracing, and
CRLF preservation mirrors `rename.go`'s established pattern with a stronger no-mixed-line-ending
assertion. The dry-run tally (76/18) was independently reproduced byte-for-byte, and two of the six
newly-skipped packages were confirmed by hand to be genuine compile-time defects the old gate would
have missed, not false skips.
