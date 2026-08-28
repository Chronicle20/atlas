# Task 16 batch `misc-b` review

Scope: six commits — `d4b753fb6` (atlas-mts), `8562ea7c4` (atlas-npc-shops),
`4796e262a` (atlas-portals), `95b3d37c7` (atlas-saga-orchestrator, data/npc +
data/portal), `9eba1cb3e` (atlas-trades/configuration), `59eab9fe1`
(atlas-transports). Brief: `.superpowers/sdd/plan/task-16-brief-misc-b.md`.
Report: `.superpowers/sdd/plan/task-16-misc-b-report.md`.

## Verdict

APPROVED

All eight review charges checked out. Evidence below.

## 1. The `Write`-over-existing-test-file incident

`git show --stat` on all six commits, individually diffed against each
commit's own parent (`git diff <c>^ <c>`), shows **zero deletions** in every
commit:

- `d4b753fb6`: `2 files changed, 45 insertions(+)`
- `8562ea7c4`: `2 files changed, 50 insertions(+)`
- `4796e262a`: `2 files changed, 42 insertions(+)`
- `95b3d37c7`: `4 files changed, 82 insertions(+)`
- `9eba1cb3e`: `2 files changed, 41 insertions(+)`
- `59eab9fe1`: `2 files changed, 44 insertions(+)`

No commit in this batch shows an unexpected deletion.

For `9eba1cb3e` (the self-reported `Write` incident on
`services/atlas-trades/atlas.com/trades/configuration/rest_test.go`):
extracted the pre-commit blob with `git show 9eba1cb3e^:...rest_test.go` and
diffed it against the committed blob. The diff is purely additive (one new
import line, one new test function appended at EOF) — every byte of the
pre-existing content is present, unchanged, in the same order. The
pre-existing file contained two test functions (not three, per the task
description's estimate — the third and further tests the task description
was likely referring to live in sibling files `model_test.go`,
`registry_test.go`, `tax_test.go` in the same package, which this commit did
not touch): `TestRestModelDecodesTheTenantsWireDocument` and
`TestWireDocumentFoldsIntoTheDomainModel`. Ran both by name plus the new
`TestTransformRoundTrip`:

```
=== RUN   TestRestModelDecodesTheTenantsWireDocument
--- PASS: TestRestModelDecodesTheTenantsWireDocument (0.00s)
=== RUN   TestWireDocumentFoldsIntoTheDomainModel
--- PASS: TestWireDocumentFoldsIntoTheDomainModel (0.00s)
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-trades/configuration	0.007s
```

Also ran the full `configuration` package test suite (`go test
./configuration/... -v`) — all tests in `model_test.go`, `registry_test.go`,
`tax_test.go` also pass, confirming no collateral damage to the package.

**Disposition: PASS.** The self-reported incident was caught and fully
corrected before commit; the committed artifact is provably byte-faithful to
the pre-existing file plus a clean addition.

## 2. `Transform` placement where `Extract` is displaced (atlas-npc-shops)

`grep -n "^func Extract\|^func Transform"` across
`services/atlas-npc-shops/atlas.com/npc/data/setup/{rest.go,processor.go}`:

```
.../data/setup/rest.go:22:func Transform(m Model) RestModel {
.../data/setup/processor.go:40:func Extract(m RestModel) (Model, error) {
```

`Transform` was added to `rest.go` as required by DOM-04; `Extract` was left
in `processor.go` untouched. Confirmed `processor.go` has zero diff across
the entire task range (`git diff c3ad6b3c4 596dbf1ad -- .../processor.go`
returns nothing). **PASS.**

## 3 & 4. Field coverage and faithfulness to `Extract`, derived independently per package

Read each package's own `model.go` and `rest.go` (and `processor.go` for
npc-shops) directly; did not accept cross-package analogy.

| Package | `Model` fields | `Transform` covers | Notes |
|---|---|---|---|
| atlas-mts/configuration | 11 (listingFee, commissionRate, commissionBase, maxActiveListings, minLevel, auctionMinHours, auctionMaxHours, fixedSaleHours, priceFloor, pageSize, minBidIncrement) | all 11 | `RestModel.Id` is `json:"-"` and `Extract` never reads it — `Transform` correctly leaves it unset. |
| atlas-npc-shops/data/setup | 11 (id, price, slotMax, recoveryHP, tradeBlock, notSale, reqLevel, distanceX, distanceY, maxDiff, direction) | all 11, **including `Id`** | Here `Extract` (processor.go:44) DOES read `m.Id`, unlike atlas-mts — `Transform` correctly populates `RestModel.Id: m.id`. Confirms the implementer did not blindly copy the "leave Id unset" pattern from a sibling. |
| atlas-portals/portal | 8 (id, name, target, portalType, x, y, targetMapId, scriptName) | all 8 | `Id` converted via `strconv.Itoa`, inverse of `Extract`'s `strconv.Atoi`. |
| atlas-saga-orchestrator/data/npc | 5 (id, name, trunkPut, trunkGet, storebank) | all 5 | Independently shaped from the portal packages — correctly not confused with them. |
| atlas-saga-orchestrator/data/portal | 8 (id, name, target, portalType, x, y, targetMapId, scriptName) | all 8 | Same shape as atlas-portals/portal and atlas-transports/data/portal by coincidence of genuine codebase duplication, but verified independently against this package's own model.go — field names, order of unexported-field access, and getter surface are specific to this file. |
| atlas-trades/configuration | 5 (taxEnabled, taxTiers []Tier, maxStagedItems, minTradeLevel, attestationTimeout) | all 5 | `taxTiers` converted `Tier -> TierRestModel` via a loop; `attestationTimeout` converted `time.Duration -> int` seconds via `int(m.attestationTimeout / time.Second)`, exact inverse of `Extract`'s `time.Duration(r.AttestationTimeoutSeconds) * time.Second`. `RestModel.Id` never read by `Extract` (rest.go:50-70 has no `Id` handling) — `Transform` correctly leaves it unset, and this asymmetry with the sibling portal-shaped packages is explicitly called out in the `Transform` doc comment, confirming this was derived from this package's own `Extract`, not copied. |
| atlas-transports/data/portal | 8 (id, name, target, portalType, x, y, targetMapId, scriptName) | all 8 | Same note as saga-orchestrator/data/portal — independently verified. |

No `Transform` populates a `RestModel` field its own package's `Extract`
never reads, except where the field genuinely round-trips through Extract
(checked per-package above). **PASS**, no borrowed field lists found.

## 5. The lossy question

Checked every `Model` field against `RestModel` for all seven packages
(table above) — every `Model` field has a corresponding `RestModel` field
that both `Extract` reads from and `Transform` writes to. No `Model` field is
silently dropped. The implementer's claim of "no `handwork-notes.md` entry
needed" is correct for this batch: no genuinely lossy `Extract` was found by
independent review.

The one asymmetric case — `RestModel.Id` in `atlas-trades/configuration`,
never read by `Extract` — is not a lossy-`Model`-field case (it's a
`RestModel`-field-`Extract`-never-consumes case, charge 4's territory) and is
correctly left unset by `Transform`, documented in the doc comment. **PASS.**

## 6. No behavior change outside the addition

`git diff <commit>^ <commit>` for the `rest.go` file in all six commits was
inspected. Every diff hunk is a pure insertion of a `Transform` function (and
in saga-orchestrator's case, two new `rest_test.go` files) — zero lines
removed or modified in any `Extract` body, any `Build()`/`With*` validation
function, or any `RestModel` struct field list. Confirmed for
`atlas-trades/configuration` specifically (the package with the most complex
`Extract`): the diff hunk is `@@ -37,6 +37,24 @@` — insertion only, `Extract`
at the bottom of the file (with its zero-fold logic and doc comments)
untouched. **PASS**, FR-17 and PRD §5 both respected.

## 7. Scope check on atlas-trades

`git show --stat 9eba1cb3e` touches only
`services/atlas-trades/atlas.com/trades/configuration/{rest.go,rest_test.go}`.
`git log --oneline -- services/atlas-trades/atlas.com/trades/data/inventory`
shows that package's `Transform` was added in Task 15 (`5e2b905ac`,
`21c22e538`), commits entirely outside this batch's range. `data/inventory`
has no diff in `9eba1cb3e`. **PASS.**

## 8. Tests actually constrain the code

Ran an independent mutation on `atlas-saga-orchestrator/data/npc/rest.go`:
changed `TrunkGet: m.trunkGet,` to `TrunkGet: 0,` in `Transform`. Re-ran
`go test ./data/npc/... -run TestTransformRoundTrip -v`:

```
--- FAIL: TestTransformRoundTrip (0.00s)
    rest_test.go:25: round trip mismatch. Expected {id:42 name:storage_npc
    trunkPut:100 trunkGet:200 storebank:true}, got {id:42 name:storage_npc
    trunkPut:100 trunkGet:0 storebank:true}
```

Test fails at the field level as expected. Reverted with `git checkout --
data/npc/rest.go`. `git status --short` on all six touched service
directories after the revert shows no residual diff (only files modified by
OTHER concurrent agents outside this batch's services — `agent-ledger.tsv`,
`progress.md`, and other batches' review artifacts — are `M`/`??`, as
expected and unrelated to this review). **PASS.**

## Module gates

Ran `go build ./... && go vet ./... && go test ./<package>/...` per touched
module (mts, npc-shops, portals, saga-orchestrator, transports); all green.
`atlas-trades/configuration` full package test suite also green (see charge
1).

## Not evaluable

None. All eight charges were fully checkable within this batch's diff
surface plus the packages' own `model.go`/`processor.go` files the diff's
correctness depends on.
