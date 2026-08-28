# Review — Task 10: W3 `transform` subcommand

Range reviewed: `b1d090265` ("chore(task-263): add Transform generation to codemod").

Note on the given range `858edb209..b1d090265`: this range actually spans two commits
(`9898af52f` then `b1d090265`). `9898af52f` ("record Task 9a review, gate PASS, and Task 10
handoff") only touches `agent-ledger.tsv`, `progress.md`, `review-task-9a.md` and was already
reviewed as part of Task 9a's gate; it is not part of this unit of work. This review covers
`b1d090265` only, which is what the brief and report describe.

## Scope

Files touched by `b1d090265`:
- `docs/tasks/task-263-backend-guideline-conformance/codemod/transform.go` (new)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/transform_test.go` (new)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` (wiring)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/classify.go` (carried-forward fix 1)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/classify_test.go` (carried-forward fix 2)

Confirmed via `git show b1d090265 --name-only` and `git diff 858edb209..b1d090265 --name-only`
(filtered) that no file outside `docs/tasks/task-263-backend-guideline-conformance/` is touched.
Step 5's dry run is genuinely dry: no file under `services/` is modified (`git status --short
services` empty per the implementer's own evidence, and no such path appears in this commit's
diff at all).

## Carried-forward fixes (both landed)

1. **`isFlatFieldValue` tightening** — `classify.go:466-511` (see `soleParamObject` at
   `classify.go:469-482`). The selector base is now resolved to `*types.Var` via
   `pkg.TypesInfo.ObjectOf`/`Uses` and compared by `types.Object` identity against `Extract`'s
   own sole parameter, not merely "some identifier." `isFlatExtract`'s signature gained a `pkg
   *packages.Package` parameter and its one call site in `ClassifyDOM04` (`classify.go:351`) was
   updated. **PASS.**
2. **`EXCLUDED-TREE` test case** — `classify_test.go:383-411` adds a `libs/atlas-packet/opcode`
   directory to `TestClassifyFILE05Inventory`'s fixture tree and asserts the `EXCLUDED-TREE`
   disposition. **PASS.**

Also a new `TestClassifyDOM04` case ("field value selector base is not the parameter",
`classify_test.go` around line 217) proves the tightened check demotes a same-shape look-alike
(`var lookalike = RestModel{...}`) from A to B2. Ran `GOWORK=off go test ./... -v` and `go vet
./...` from the codemod module: all suites pass, no vet output.

## Finding 1 (BLOCKING) — the type-check gate never actually type-checks the generated test file

`transform.go:517-543` (`typeChecksWithOverlay`) builds a `packages.Config` with `Mode`, `Dir`,
`Env`, and `Overlay`, but **no `Tests: true`**:

```go
cfg := &packages.Config{
    Mode:    mode,
    Dir:     pkgDir,
    Env:     loadEnv(),
    Overlay: overlay,
}
pkgs, err := packages.Load(cfg, ".")
```

`golang.org/x/tools/go/packages` only loads `_test.go` files — including in-package ones like
`rest_test.go` — when `Tests: true` is set; otherwise they are excluded from the loaded package
entirely, overlay or not. `rename.go:279-285` (`loadModuleForRename`) already documents this
exact fact in the same codebase ("loads root's package graph including its internal and external
test variants (`Tests: true`)... classify's existing per-root loading (which does not need test
variants) is unaffected") — so the omission in `transform.go` is not an obscure gap, it is the
one place in this commit where the precedent from `rename.go` needed to be followed and wasn't.

I verified this empirically rather than by inference: I added a temporary test
(`docs/tasks/task-263-backend-guideline-conformance/codemod/zzscratch_check_test.go`, removed
before finishing — `git status --short` on the codemod directory is clean) that calls
`typeChecksWithOverlay` against a scratch module whose `lib_test.go` contains
`return "not an int"` from a function declared to return `int` — an unambiguous type error.
`typeChecksWithOverlay` returned `nil`:

```
=== RUN   TestZZScratchOverlayTestsFlag
    zzscratch_check_test.go:16: typeChecksWithOverlay err = <nil>
--- PASS: TestZZScratchOverlayTestsFlag (0.01s)
```

Consequence: `GenerateTransform` (`transform.go:63-144`) builds `newRestTestGo` and puts it in
the overlay at `restTestGoPath` (`transform.go:135-141`), but that overlay entry is inert — the
gate only ever proves `rest.go`'s new `Transform` function compiles. A generated
`TestTransformRoundTrip` with a wrong `Set<Field>` argument type, an unresolved import, a bad
builder call, or any other test-file-only type error would sail through "type-checked" and be
written to 82 real packages' `rest_test.go` with **zero verification** — exactly the risk Step 5
of the algorithm exists to close ("The codemod never writes code it has not checked"). This is
the task's own top-priority check (#1 in the dispatch brief) and it fails: the scratch/overlay is
only partially type-checked, and the untested half is the one whose entire purpose is catching
R1 (wrong-sibling field mapping).

This also explains why none of `TestGenerateTransform`'s 7 cases caught it: the one
type-check-failure fixture (`transform_test.go:169-191`) induces its error via `rest.go`'s
`Transform` body (an illegal `[]string`→`int` conversion), never via `rest_test.go`. No fixture
in this commit exercises a test-file-only type error, so this gap was not merely unfixed, it was
untested in a way that would have surfaced it.

**Action needed:** add `Tests: true` to `typeChecksWithOverlay`'s `packages.Config` (and confirm
`pkgs` selection still resolves the right variant — with `Tests: true`, `packages.Load` returns
both the non-test and test-augmented package for `"."`; the code will need to pick the
test-augmented one, or check errors across all returned packages), plus a fixture that puts the
type error only in the generated round-trip test to prove the fix.

## Finding 2 (non-blocking) — no fixture proves cross-field distinctness within one type

The round-trip test's whole reason for existing is catching a field mapped to the wrong
same-typed sibling (R1). `deriveTestValues` (`transform.go:267-313`) computes `n := i+1` over the
*global* field index and multiplies by 11 for integers — which is correct and does yield distinct
values for two integer fields — but no fixture in `TestGenerateRoundTripTest` (`transform_test.go:238-369`)
has two fields of the same type to assert this concretely (e.g. two `uint32` fields → `11` and
`22`, not `11` and `11`). The existing fixtures only ever have one field per type, so a
hypothetical regression that reset `n` per type, or that always used `n*11` for the *first*
integer field regardless of position, would not be caught by any test in this commit. Recommend
adding a two-integer-field (or two-string-field) case before Task 11 starts consuming this
generator at scale.

## Finding 3 (non-blocking) — line-ending normalization not preserved on the merge path

`mergeIntoRestGo` (`transform.go:449-456`) and `mergeIntoRestTestGo` (`transform.go:471-510`) run
the merged source through `format.Source`/`format.Node`, which always emits LF line endings.
`rename.go`'s disk-write path (`rename.go:255-260`, `bytes.Contains(orig, []byte("\r\n"))`)
explicitly restores CRLF when the original file used it, per CLAUDE.md's "preserve existing line
endings" rule; `transform.go`'s merge functions have no equivalent step. `grep -rlUP '\r\n'
--include=rest.go services` found no CRLF `rest.go` files in the tree today, so this is not
exploited by any of the 94 tier-A packages, but it is a real gap relative to a rule this same
repository already enforces elsewhere in the codemod, and it would silently normalize any CRLF
`rest.go`/`rest_test.go` introduced later.

## Verified as correct

- **No real file is ever mutated by a failing `GenerateTransform` call.** `GenerateTransform`
  only reads `rest.go` from disk (`transform.go:116`) and builds `newRestGo`/`newRestTestGo` in
  memory; `os.WriteFile` is only reached from `writeTransformedFiles` (`transform.go:548-578`),
  which `transformRepo` calls only when `!dryRun` and only after `GenerateTransform` has already
  returned success (`transform.go:693-707`). `TestGenerateTransform`'s skip cases assert
  `rest.go` on disk is byte-identical to the input after a skip (`transform_test.go:213-219`) —
  confirmed by running the suite.
- **Value-derivation table matches the brief exactly**, including the global (not per-type) field
  index — verified against the brief's own worked example (`A uint32`→`11`, `B string`→`"field2"`)
  in `TestGenerateRoundTripTest/builder_available` (`transform_test.go:312-317`), and confirmed by
  test run.
- **`two or more bool fields` skip** (`transform.go:296-300`) and **`unsupported field type`
  skip** (`transform.go:291-294`, `308-309`) are both implemented and covered by
  `TestGenerateTransform/two_bools` and `/slice_field`.
- **No partially-zero-valued test can be emitted**: `deriveTestValues` returns an error (not a
  partial slice) the moment any field's type is unsupported or a second bool is seen
  (`transform.go:293`, `298-299`, `309`), and `GenerateTransform` propagates that error before
  ever calling `buildRoundTripTestFunc` (`transform.go:92-95`) — there is no path that reaches
  test generation with an incomplete `values` slice.
- **`RestModel` fields `Extract` does not read are never emitted** — `fieldMappingsFromExtract`
  only walks `Extract`'s own composite-literal keys (`transform.go:213-255`); confirmed by
  `TestGenerateTransform/unmapped_RestModel_field` explicitly asserting no `Id:` key
  (`transform_test.go:226-228`).
- **Pointer-returning `Extract`** is handled: `Transform` is always `func Transform(m Model)
  (RestModel, error)` (`transform.go:378`), and the round-trip comparison dereferences
  (`*got`) when `Extract` returns `*Model` (`transform.go:419-422`), avoiding the
  type-correct-but-always-false `*Model`-vs-`Model` comparison bug. Covered by
  `TestGenerateTransform/pointer_return`.
- **Builder usability is resolved via `go/types`**, not string-matching: `builderConstructor`
  requires a zero-arg `NewBuilder` (`transform.go:319-339`), `builderCoversAllFields` requires a
  `Build` method and, for every mapped field, a one-arg `Set<Field>` resolved via
  `types.LookupFieldOrMethod` (`transform.go:344-361`). Covered by all three
  `TestGenerateRoundTripTest` cases, including "builder missing a setter" correctly falling back
  to the composite-literal form.
- **No `*_testhelpers.go` file is ever emitted** — the generated test is always merged into
  `rest_test.go` (`transform.go:125`, `565`); no other file name appears anywhere in the write
  path.
- **`main.go` wiring** (`main.go`, diff in `b1d090265`) registers `"transform": runTransform`
  alongside the pre-existing `classify`/`rename` entries; the doc comment above the map was
  updated to drop Task 10 from the "still to land" list. Confirmed by reading the diff.
- **`-repo`/`-classification`/`-ledger`/`-only`/`-dry-run` flags** (`transform.go:585-645`) match
  the brief's Step 3 list and mirror `rename.go`'s existing `-only` convention
  (comma-separated repo-relative prefixes).
- **`GOWORK` is stripped from the load environment** (`load.go:15-25`, `loadEnv`, unchanged by
  this commit but relied on by `typeChecksWithOverlay`), consistent with the module-local
  (`GOWORK=off`) execution model the brief specifies.

## TDD-honesty ruling (the implementer's self-reported concern)

The implementer's own report states RED was not captured as a literal failing run —
`transform.go` and `transform_test.go` were written together. Setting that process point aside,
the tests as written **are** genuinely discriminating for everything they assert: each skip case
requires both a specific `*skipError` reason substring and (for skip cases) a byte-identical
`rest.go`; each generation case requires specific rendered substrings that a plausibly wrong
implementation (wrong field, wrong order, wrong conversion type, missing dereference) would not
produce. I did not need to hypothesize this — Finding 1 above is direct proof that this test
suite *is* capable of exposing a real implementation gap when a fixture actually exercises the
gap; the problem is that the type-check-gate defect the suite should have caught (a test-file-only
type error) fell into a fixture blind spot common to all 7 `TestGenerateTransform` cases, not that
the assertions themselves are weak. So: RED being implicit rather than a captured run is a process
gap worth noting, but it is not why Finding 1 slipped through — the missing fixture shape is. I
would not require a literal stub-and-rerun redo on process grounds alone; I would require the
fixture gap in Finding 2/the Finding-1 fix to be closed.

## Not evaluable

- The real dry-run's 82 APPLIED / 12 SKIPPED counts and the specific claim about
  `atlas-npc-conversations/petdata` are taken from the implementer's report; I did not re-run the
  dry run against the full tree in this review (out of scope for a per-unit review — `services/`
  is read-only input the codemod touches only via `-dry-run`, and re-running it would not change
  the disposition of this commit).
- `progress.md`'s Task 10 completion entry (mentioning `b1d090265` and "DONE_WITH_CONCERNS") is
  present as an uncommitted working-tree change at review time. This is expected per the brief's
  own Step 6 (only `codemod/` is committed) and is not part of this commit; not evaluated as part
  of `b1d090265`.

## Verdict

CHANGES_REQUIRED — Finding 1 is a genuine defect in the exact mechanism the task asked this
review to verify first: the overlay type-check gate silently never checks the generated test
file, which is the artifact whose only job is catching R1. This must be fixed (and covered by a
fixture that puts the type error only in the generated test) before Task 11 starts applying
`transform` to real packages, since every package the gate wrongly approves ships an unverified
`rest_test.go` 82 times over.
