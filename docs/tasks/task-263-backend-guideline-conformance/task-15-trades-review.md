# Task 15 review — batch `atlas-trades`

Reviewed commits: `5e2b905ac` (feat, initially overwrote pre-existing test file),
`21c22e538` (fix, restores dropped tests).

Brief: `.superpowers/sdd/plan/task-15-brief-atlas-trades.md` +
`.superpowers/sdd/plan/task-15-common.md`.
Report: `.superpowers/sdd/plan/task-15-report-atlas-trades.md`.

## Primary risk: did the fix commit actually restore the overwritten tests byte-faithfully?

**Verified — the implementer's claim holds.**

```
$ git diff 6c840dba1..21c22e538 --stat -- services/atlas-trades/atlas.com/trades/data/inventory/
 .../data/inventory/rest.go        | 30 ++++++++++++++
 .../data/inventory/rest_test.go   | 48 ++++++++++++++++++++++
 2 files changed, 78 insertions(+)
```

- `rest_test.go`: diffing pre-existing commit `6c840dba1` against the final `21c22e538` shows a
  pure hunk-append — three new imports (`reflect`, `uuid`, `slot`, `item`) and one new function
  (`TestTransformRoundTrip`, lines 142-184) appended after the existing final test
  (`TestAssetsIsNotWritableThroughTheGetter`). No line inside any of the five original test
  functions (`TestExtractCarriesAssetAttributes`, `TestFindBySlotResolvesANegativeSlot`,
  `TestFindBySlotMissesAnEmptySlot`, `TestFindByIdResolvesAcrossSlots`,
  `TestAssetsIsNotWritableThroughTheGetter`) was touched — confirmed by running all five plus the
  new test and getting 6/6 PASS (see Live test run below).
- `rest.go`: pure append of `TransformAsset`/`Transform` after the existing `Extract`/`ExtractAsset`
  (`services/atlas-trades/atlas.com/trades/data/inventory/rest.go:145-172`), 30 insertions, 0
  deletions.
- No other file in the package (`model.go`, `processor.go`, `requests.go`, `mock/`) was touched by
  either commit — confirmed by `git diff 6c840dba1..21c22e538 --stat` scoped to the whole
  `data/inventory/` directory and separately to all of `services/atlas-trades/`, both showing the
  identical two-file, 78-insertion, 0-deletion result. No other pre-existing file was silently
  overwritten anywhere in the package.

Verdict on the primary risk: PASS. The net diff across the two commits is genuinely additive; the
five original assertions are present with their original bodies, not merely same-named stand-ins.

## Field-by-field mapping check

`services/atlas-trades/atlas.com/trades/data/inventory/model.go`:

- `Asset{id, slot, templateId, quantity, flag}` — all 5 fields mapped in `TransformAsset`
  (`rest.go:146-154`), matching `ExtractAsset` (`rest.go:128-130`) field-for-field, 1:1, no
  omissions.
- `Model{id, inventoryType, capacity, assets}` — all 4 fields mapped in `Transform`
  (`rest.go:157-173`), matching `Extract` (`rest.go:133-143`) field-for-field. No embedded type in
  either struct.
- `RestModel{Id, InventoryType, Capacity, Assets}` and `AssetRestModel{Id, Slot, TemplateId,
  Quantity, Flag}` — every field of both wire structs is consumed by the corresponding `Extract*`
  and produced by the corresponding `Transform*`. No field is orphaned on either side.

This package is genuinely narrower than the near-identical `atlas-inventory` compartment/asset
types (per `model.go:1-14`'s package doc: the projection deliberately omits the ~30 equipment
stat fields atlas-inventory carries). I did not assume equivalence with atlas-inventory; I
enumerated `atlas-trades`' own `model.go` and `rest.go` directly and confirmed the `Transform`
bodies against that enumeration, not against another package's shape.

No hardcoded-value accessor exists on `Asset` or `Model` (all five/four accessors read a stored
field; none return a literal). Confirmed neither `Transform` emits anything beyond what `Extract`
reads.

## Live mutation (independent of the implementer's proof)

The implementer's own mutation proof (in their report) dropped the `Flag` field. Per the review
brief, I mutated a field their proof did **not** touch: `InventoryType` in `Transform`.

```go
// mutated:
InventoryType: inventory.Type(0),   // was: InventoryType: m.Type(),
```

```
$ go test ./data/inventory/... -run TestTransformRoundTrip -v
=== RUN   TestTransformRoundTrip
=== RUN   TestTransformRoundTrip/Asset
=== RUN   TestTransformRoundTrip/Model
    rest_test.go:181: round trip mismatch: got {... inventoryType:0 ...}, want {... inventoryType:2 ...}
--- FAIL: TestTransformRoundTrip (0.00s)
    --- PASS: TestTransformRoundTrip/Asset (0.00s)
    --- FAIL: TestTransformRoundTrip/Model (0.00s)
FAIL
```

Field-level failure confirmed (the `Asset` subtest correctly still passes since it doesn't touch
`InventoryType`; only `Model` fails, with the exact field named in the diff). Reverted from a
saved copy of `rest.go`, then:

```
$ git diff --exit-code -- services/atlas-trades/atlas.com/trades/data/inventory/rest.go
CLEAN
$ go test ./services/atlas-trades/atlas.com/trades/data/inventory/... -v
--- PASS: TestExtractCarriesAssetAttributes
--- PASS: TestFindBySlotResolvesANegativeSlot
--- PASS: TestFindBySlotMissesAnEmptySlot
--- PASS: TestFindByIdResolvesAcrossSlots
--- PASS: TestAssetsIsNotWritableThroughTheGetter
--- PASS: TestTransformRoundTrip (incl. /Asset, /Model)
ok  	atlas-trades/data/inventory	(cached)
```

All 6 tests pass (5 original + 1 new), file byte-identical after revert.

## Fixtures

`TestTransformRoundTrip/Asset`: `NewAsset(101, slot 3, item 2000001, qty 7, flag 0x02)` — all
non-zero, non-default.
`TestTransformRoundTrip/Model`: two distinct assets (ids 101/102, slots 3/4, items
2000001/2000002, quantities 7/9, flags 0x02/0x04), `inventory.Type(2)`, `capacity 24`, random
`uuid.New()` id. All fields distinct and non-zero across both subtests. PASS.

## Extract/Transform inventory completeness

```
$ grep -rn '^func Extract\|^func Transform' services/atlas-trades/atlas.com/trades/data/inventory/
rest.go:128:func ExtractAsset(rm AssetRestModel) (Asset, error) {
rest.go:133:func Extract(rm RestModel) (Model, error) {
rest.go:146:func TransformAsset(a Asset) (AssetRestModel, error) {
rest.go:157:func Transform(m Model) (RestModel, error) {
```

Grep run over the whole package directory (not just `rest.go`) per the common brief's requirement
— no `Extract*` exists outside `rest.go` in this package, and both `Extract*` functions listed in
the per-service brief now have exact-named inverses. No extra pairs, no missing pairs, no
pre-existing `Transform*` (matches the report's claim).

## Docs / exemption note

```
$ git diff 6c840dba1..21c22e538 --stat -- docs/
(empty)
```

No file under `docs/` was touched by either commit. No exemption note was added to
`handwork-notes.md` (this package has a `RestModel`, so none was required per the common brief).

## Gate

```
$ tools/lint.sh --check --fmt --go services/atlas-trades/atlas.com/trades
lint.sh: OK
```

`go build`/`go vet`/`go test ./...` at module root reported clean in the implementer's report
(pasted transcript, `6/6 tests`); I independently re-ran the package-level test suite above and
observed the same 6/6 pass.

## Not evaluable

None. The full review surface (both commits, the touched package) was covered by direct diff,
field enumeration, and an independent live mutation.

## Disposition

All required checks pass:
1. Primary risk (silent test-file overwrite) — verified false; net diff is genuinely additive,
   original five tests byte-faithful.
2. `git diff --stat` shows zero deletions in the package across both commits; no other file was
   overwritten.
3. Field enumeration for `Model`/`RestModel`/`Asset`/`AssetRestModel` — complete, no gaps, no
   embedded types, no equivalence assumption borrowed from atlas-inventory.
4. No hardcoded-value accessors present.
5. Independent live mutation (a field the implementer's own proof did not touch) produced a
   field-level test failure, and the revert is confirmed clean via `git diff --exit-code`.
6. Fixtures are distinct, non-default, non-zero.
7. No `docs/` file touched, no exemption note added.
8. Both `Extract*` functions have inverses; grep over the whole package confirms no orphan.

No blocking findings. No non-blocking findings.
