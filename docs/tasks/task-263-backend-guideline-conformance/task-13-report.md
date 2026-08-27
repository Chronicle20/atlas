# Task 13 report — W3 hand work: the four packages named in #1498 (FR-7)

## Summary

Implemented `Transform*` functions (D1, D2, FR-3) for the three
`NO-RESTMODEL` packages named in the brief and verified/completed the
round-trip test for the one tier-A package already touched by the codemod.

## Files changed

1. `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go`
   — added a `setFields(bool, int32, bool)` pointer-receiver method to each
     of the five wire types (`EquipmentRestModel`, `ConsumableRestModel`,
     `SetupRestModel`, `EtcRestModel`, `CashRestModel`), a generic
     `transform[R any, PR interface{ *R; setFields(...) }](m Model) R`
     helper mirroring the existing `extract[R]`, and the five named
     one-liner functions `TransformEquipment`, `TransformConsumable`,
     `TransformSetup`, `TransformEtc`, `TransformCash`.
2. `services/atlas-channel/atlas.com/channel/data/tradeability/rest_test.go`
   — new file, `TestTransformRoundTrip` table-driven over the five
     compartments, each asserting `reflect.DeepEqual(extract(Transform*(m)), m)`
     with the exact input values from the brief's table.
3. `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go`
   — same shape as (1), but the two-field variant (`fields()/setFields()`
     take only `(bool, int32)` since this package's `Model` has no `only`
     field).
4. `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest_test.go`
   — new file, same table-driven round-trip shape, using `NewModel(bool, int32)`.
5. `services/atlas-channel/atlas.com/channel/monsterbook/rest.go`
   — added `TransformCard(Card) (CardRestModel, error)` (inverse of the
     existing `ExtractCard`) and `Transform(Collection) (CollectionRestModel, error)`
     (inverse of the existing `Extract`).
6. `services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go`
   — added `TestTransformRoundTrip` with two subtests, `card` and
     `collection`, each asserting `reflect.DeepEqual` round trip with every
     field set to a distinct non-zero value.
7. `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest_test.go`
   — changed the existing `TestTransformRoundTrip`'s `monsterId` value from
     `22` to `100100` to match the brief's exact table value. The `Transform`
     function itself (`rest.go:45-50`) was already present from Task 12 and
     was verified correct: it maps only `monsterBook`/`monsterId` (mirroring
     `Extract`) and does not emit `Id`, since `Extract` does not read it
     either.

## Ambiguity resolution encountered (brief-flagged)

- `atlas-monster-book/.../data/consumable`: **Task 12 had already generated
  a correct `Transform`.** I verified it against `Extract` field-by-field
  (both map only `monsterBook`/`monsterId`; `RestModel.Id` is untouched by
  either direction) and did not regenerate it. I only adjusted the existing
  round-trip test's literal value to match the brief's exact `100100`.

## Discrepancy noted (not a defect — brief prose vs. actual code)

The brief's table for `monsterbook`'s `collection` subtest says "a
`Collection` holding two distinct `Card`s." The actual `Collection` struct
(`monsterbook/processor.go:21-29`) has no `[]Card` field — it is a flat
summary (`bookLevel`, `normalCount`, `specialCount`, `totalUniqueCards`,
`coverCardId`, `coverMonsterId`, `expBonusPercent`). The `[]Card` slice
lives on the separate `Model` type (`model.go:13`), which this task's
`Transform`/`Extract` pair does not touch. I built the `collection` subtest
per the brief's actual normative requirement — "every field set to a
distinct non-zero value" — using `Collection`'s real fields, since that is
what `Transform`/`Extract` operate on.

## TDD evidence

Tests and implementation were written together per file rather than in a
strict red/green sequence captured as separate commands (the `Transform`
shape was mechanically derived from the sibling `extract`/`Extract`
functions already present, so there was no ambiguity to de-risk with a
failing run first). Correctness was confirmed by running each new test
after implementation; all pass. Below is the passing evidence per package.

### atlas-channel — `data/tradeability`

```
$ go test ./data/tradeability/... -run TestTransformRoundTrip -v
=== RUN   TestTransformRoundTrip
=== RUN   TestTransformRoundTrip/equipment
=== RUN   TestTransformRoundTrip/consumable
=== RUN   TestTransformRoundTrip/setup
=== RUN   TestTransformRoundTrip/etc
=== RUN   TestTransformRoundTrip/cash
--- PASS: TestTransformRoundTrip (0.00s)
    --- PASS: TestTransformRoundTrip/equipment (0.00s)
    --- PASS: TestTransformRoundTrip/consumable (0.00s)
    --- PASS: TestTransformRoundTrip/setup (0.00s)
    --- PASS: TestTransformRoundTrip/etc (0.00s)
    --- PASS: TestTransformRoundTrip/cash (0.00s)
PASS
ok  	atlas-channel/data/tradeability	0.005s
```

### atlas-channel — `monsterbook`

```
$ go test ./monsterbook/... -run TestTransformRoundTrip -v
=== RUN   TestTransformRoundTrip
=== RUN   TestTransformRoundTrip/card
=== RUN   TestTransformRoundTrip/collection
--- PASS: TestTransformRoundTrip (0.00s)
    --- PASS: TestTransformRoundTrip/card (0.00s)
    --- PASS: TestTransformRoundTrip/collection (0.00s)
PASS
ok  	atlas-channel/monsterbook	0.008s
```

### atlas-inventory — `data/tradeability`

```
$ go test ./data/tradeability/... -run TestTransformRoundTrip -v
Go test: 6 passed in 2 packages
```
(rtk-condensed output; all 5 subtests + parent passed, no FAIL.)

### atlas-monster-book — `data/consumable`

```
$ go test ./data/consumable/... -run TestTransformRoundTrip -v
Go test: 1 passed in 1 packages
```

## Module-local gates (Step 5)

Ran from each module root: `go build ./... && go vet ./... && go test ./...`

- `services/atlas-channel/atlas.com/channel` — build clean, vet clean,
  `go test ./...` produced no `FAIL` lines across the full package tree.
- `services/atlas-inventory/atlas.com/inventory` — build clean, vet clean,
  `go test ./...` produced no `FAIL` lines.
- `services/atlas-monster-book/atlas.com/monster-book` — build clean, vet
  clean, `go test ./...` produced no `FAIL` lines.

## Self-review

- `Transform`/`extract` inverses checked field-by-field against the
  existing `Extract`/`fields()` for both tradeability packages; no field
  dropped or renamed.
- The generic `transform[R, PR]` helper follows the existing repo idiom of
  a pointer-type-parameter constraint (`PR *R` with a method set) to build
  a zero-value `R` generically — this is the standard Go pattern for
  "generic constructor," and keeps the five named functions one-liners per
  FR-3's explicit preference.
- `monsterbook.TransformCard`/`Transform` read unexported fields directly
  (same package) per D1.
- No `Id` fields are populated by any of the new `Transform*` functions in
  `data/tradeability`, matching the fact that `Model` in both packages
  carries no `Id` and `extract`/`fields()` never read `RestModel.Id`
  either — Transform is the exact structural inverse of Extract, nothing
  more, nothing invented.
- Did not touch `services/atlas-storage/atlas.com/storage/asset` (carried
  forward as already conformant) or any package outside the brief's `###
  Files` list.
- Git history: three commits, one per service, exactly matching Step 6's
  specified `git add`/`git commit` invocations, each using explicit paths
  (no `git add -A`/`.`).

## Issues or concerns

None. All three module gates pass; the fourth package (`atlas-monster-book`
consumable) required only a test-value correction, not new production code.

## Commits

- `22bb8c1` — `feat(atlas-channel): add Transform for tradeability and monsterbook`
- `39c6b08` — `feat(atlas-inventory): add Transform for tradeability`
- `19e500d` — `feat(atlas-monster-book): add Transform round-trip test for consumable`

## Follow-up: gofumpt gate fix (2026-08-26)

### Failure

`tools/verify.sh --quick --base d0ba529db` failed the `lint & format guard`:
`lint.sh: FMT FAIL — services/atlas-channel/atlas.com/channel` (gofumpt wants
a blank line between consecutive one-line method declarations in
`data/tradeability/rest.go`).

### Scope checked

`git diff --name-only d0ba529db..HEAD -- '*.go'` — 15 files across
atlas-cashshop, atlas-channel (data/tradeability, monsterbook), atlas-drops,
atlas-inventory, atlas-messengers, atlas-monster-book, atlas-parties.

### Fix

Ran the guard's own formatter (`tools/lint.sh --fmt --go <module>`, i.e.
`golangci-lint fmt -c .golangci.yml ./...`, the same invocation
`lint.sh`'s FMT layer uses — not a bare `gofumpt` binary, which was not on
PATH) per affected module root, then re-verified with
`tools/lint.sh --check --fmt --go <module>` until each printed `lint.sh: OK`.

Two files needed a rewrite (both `TradeBlock/TradeAvailable[/Only]`
`setFields` one-liner runs where inserting a blank line before the last
method in the run also changed the column-alignment of the sibling
one-liners above it, requiring a second `--fmt` pass to converge):

- `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go`
- `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go`

The other 13 changed files (`rest_test.go` files, `monsterbook/rest.go`,
and the four batch-A packages under atlas-cashshop, atlas-drops,
atlas-messengers, atlas-parties, atlas-monster-book) were already
gofumpt-clean — `tools/lint.sh --check --fmt --go <module>` on each of the
7 affected module roots (atlas-cashshop, atlas-channel, atlas-drops,
atlas-inventory, atlas-messengers, atlas-monster-book, atlas-parties)
returned `lint.sh: OK` (exit 0) after the two-file fix, confirming no other
file in scope needed a rewrite.

Each diff is a mechanical blank-line insertion (plus alignment respacing
in atlas-inventory, since gofumpt re-aligns comment/brace columns across a
contiguous run of one-line methods once one member of the run gains a
blank-line separator) — no behavior change.

### Verification

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test ./...
  → ok, all packages pass (data/tradeability: 17 passed in 2 packages)

cd services/atlas-inventory/atlas.com/inventory && go build ./... && go vet ./... && go test ./...
  → Go test: 135 passed in 39 packages
```

`tools/lint.sh --check --fmt --go` re-run against all 7 affected module
roots together: `lint.sh: OK` (exit 0).

`tools/verify.sh` was not run — the gate runs outside this agent per task
instructions.

### Files gofumpt rewrote

- `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go`
- `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go`

No other file was touched.

### Commit

`9eb0a2f12` — `style(task-263): gofumpt the Task 13 and 14a hand-written Transform files`

Note: during this fix, `git status` showed unstaged changes to
`docs/tasks/task-263-backend-guideline-conformance/agent-ledger.tsv` and
`progress.md`, and a new untracked `task-14a-review.md`, from a concurrent
session sharing this worktree. Those were left untouched and not staged —
only the two `rest.go` paths were added and committed by name.
