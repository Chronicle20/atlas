# Task 21 review — `atlas-maker` eligibility evaluation

Commit range: `06b48f3..ea58e0781` (single commit `ea58e0781`).

## Verdict: APPROVED_WITH_FINDINGS

## 1. Scope extension (`compartment/model.go`, `compartment/rest.go`)

**Right call, and independently verified correct.**

- Wire field: `services/atlas-inventory/atlas.com/inventory/asset/rest.go:10` —
  `Slot int16 \`json:"slot"\``. The implementer's `assetRestModel.Slot int16
  \`json:"slot"\`` (`compartment/rest.go`) matches name and type exactly.
- Sign convention: `services/atlas-inventory/atlas.com/inventory/compartment/processor_accommodation.go:22-27`
  — `freeSlots` comment: "Equipped items live in negative slots and never
  consume inventory space" and the code only counts `a.Slot() >= 1` as
  occupied. `Snapshot`'s `Equipped` (`craft/snapshot.go:51-57`) marks an item
  equipped only when `invType == inventory.TypeValueEquip && a.Slot() < 0` —
  matches the sign convention exactly. No corruption risk for Task 23's
  per-slot destruction (which only ever walks `Slots()`, itself built from
  the same decoded field).
- The brief's `Slots(item.Id) []SlotHolding` contract was genuinely
  unbuildable from Task 19's pre-existing `AssetModel` (no slot field). This
  is "finish producible work," not scope creep: the fix lives exactly where
  the wire decode already lives (`compartment/model.go`,
  `compartment/rest.go`), is additive (new private field + accessor + two
  new exported constructors), and the existing `compartment` test suite
  passes unchanged (verified: `go test ./compartment/... -count=1` → PASS).

## 2. `Builder` pattern

**Follows the repo convention; not test-only scaffolding leaking into
production code.**

- `reagent/builder.go:49-86` and `crystalband/builder.go` establish
  `NewBuilder(...) *Builder`, chained `Set*(...) *Builder`, `Build()` —
  `compartment/model.go`'s new `Builder` (`NewBuilder`, `SetCapacity`,
  `AddAsset`, `Build()`) matches that shape. Minor stylistic difference:
  `reagent.Builder.Build()` returns `(Model, error)` and validates; the new
  `compartment.Builder.Build()` returns `Model` with no validation. Not a
  defect — `compartment.Model`'s fields have no analogous "invalid stat"
  class of constraint to validate — but worth naming so a later Builder in
  this package doesn't silently assume the no-validation shape is the norm.
- Confirmed usage is test-only, same as the two precedents: `grep -rln
  "reagent.NewBuilder"` → only `reagent/processor_test.go`,
  `reagent/builder_test.go`; `compartment.NewBuilder`/`NewAssetModel` →
  only `craft/eligibility_test.go`, `craft/snapshot_test.go`. Exported
  production code used exclusively by tests, not a `*_testhelpers.go` file —
  matches the mandated pattern, not a workaround of it.

## 3. Judgment call A — accommodation for `randomReward` recipes

`awardsOf` (`craft/eligibility.go:187-201`) checks `CanAccommodate` against
*every* possible reward's `(ItemId, ItemNum)` when a recipe has a
`randomReward` list, even though only one is drawn at craft time.

Design §4.2.2 step 6 says "Free-slot capacity for every award (FR-3.6)";
FR-3.6 itself says "for every awarded item" — neither the PRD nor design.md
pins whether that means "every item actually granted in one craft" (plural
mainly for disassemble's crystals-plus-refund case, design.md:523) or "every
item that could possibly be drawn." The implementer's reading is the
stricter of the two plausible readings.

**Assessment: defensible and self-consistent, but conservative in a way that
can produce false `inventory_full` results.** If Task 23/24's actual craft
execution checks accommodation only for the *specific* reward drawn (which
is the natural reading of "only one reward is drawn at craft time"), a
character who has room for every reward *except* one low-probability one
would be marked ineligible here even though most draws would succeed. This
is fail-safe (never over-permits), not fail-unsafe (never corrupts state or
under-consumes), so it is not blocking — but it is a real behavioral choice
that Task 23/24 must either ratify or override before the two units'
semantics can be said to agree. Flagging per the implementer's own request
for controller ruling before Task 23.

## 4. Judgment call B — `missing_prerequisite_quest`

Confirmed independently:
- **Present**: `design.md:376` ("`missing_prerequisite_quest` (422, from
  C-5)"), `plan.md:3151`, `context.md:136`.
- **Absent** from `prd.md`'s §5 error table, read directly at
  `prd.md:224-235` (six 422 codes: `level_too_low`, `skill_level_too_low`,
  `insufficient_materials`, `missing_prerequisite_item`,
  `insufficient_mesos`, `inventory_full`, plus disassemble-only
  `equip_not_found`/`no_crystal_mapping` and `recipe_not_found`).

The design/plan correction is binding per this branch's own stated rule
(`plan.md:15`: "The design's §2 corrections are binding and override the
PRD"). Following it is correct.

**1:1 mapping, enumerated both directions:**

| Reason constant (`craft/eligibility.go:22-28`) | PRD §5 / design C-5 code | In scope for `craft`? |
|---|---|---|
| `ReasonLevelTooLow` | `level_too_low` | yes |
| `ReasonSkillLevelTooLow` | `skill_level_too_low` | yes |
| `ReasonInsufficientMaterials` | `insufficient_materials` | yes |
| `ReasonMissingPrerequisiteItem` | `missing_prerequisite_item` | yes |
| `ReasonMissingPrerequisiteQuest` | `missing_prerequisite_quest` (design C-5) | yes |
| `ReasonInsufficientMesos` | `insufficient_mesos` | yes |
| `ReasonInventoryFull` | `inventory_full` | yes |

Not produced by `craft`, correctly out of scope for eligibility evaluation:
`recipe_not_found` (404, recipe lookup — Task 23/24's endpoint concern per
the report, correctly deferred), `equip_not_found` / `no_crystal_mapping`
(422, disassemble-specific, a different FR path). No orphan on either side
within this unit's scope.

## 5. Check order

Verified against `craft/eligibility.go:110-185`, matches the brief's six
steps exactly:
1. `reqLevel` (line 118) then `reqSkillLevel` floored at 1 (lines 121-132).
2. `reqItem`/`reqEquip` against the snapshot (lines 135-140).
3. `reqQuest`, conditional on `len(reqs) > 0` (lines 145-159).
4. Materials summed across slots (lines 162-166).
5. `meso` (lines 169-171).
6. `CanAccommodate` (lines 176-182).

The quest skip is genuine, not issued-and-ignored: `TestReqQuestIsOnlyReadWhenTheRecipeCarriesOne`
(`craft/eligibility_test.go:337-357`) asserts `h.questCallCount == 0` via a
mock that increments on every call — a call that returned an empty slice
would still increment the counter and fail this assertion. Independently
re-ran: `go test ./craft/... -run TestReqQuestIsOnlyReadWhenTheRecipeCarriesOne -v`
→ PASS.

## 6. Maker skill constants

`craft/eligibility.go:50-55` uses exactly the four brief-named constants
(`skillconst.BeginnerMaker`, `NoblesseMaker`, `LegendMaker`, `EvanMaker`)
from `libs/atlas-constants/skill`. No service-local list. `grep -rn
"BeginnerMaker\|NoblesseMaker\|LegendMaker\|EvanMaker"` under `services/atlas-maker`
shows no redefinition, only import and use.

## Carried finding from Task 20 — cache mutation

**Verified clean.** `recipe.Model.Materials()`, `.RandomRewards()`, and
`.QuestRequirements()` (`recipe/model.go:94-108`) return the model's own
backing slices with no defensive copy, as flagged in the Task 20 review.
`craft/eligibility.go` only `range`s over them (lines 154, 162, 197) and
reads `mat.ItemId`/`mat.Count`, `req.QuestId`/`req.State`,
`rw.ItemId`/`rw.ItemNum` — `Material`, `QuestRequirement`, and `Reward` are
value types, so the range variables are copies; no field of any element is
ever assigned through. No write path exists in this unit's diff into any of
these slices or their elements. The shared per-tenant cache is not at risk
from this unit.

## Test honesty

Spot-checked `TestEligibilityOnePerExclusionReason`
(`craft/eligibility_test.go:188-290`): each of the 10 subtests asserts
`assert.Equal(t, tc.reason, e.Reason)` in addition to `assert.False(t,
e.Eligible)` (line 287) — not merely `Eligible == false`. A regression that
returned the wrong reason for the right exclusion would be caught.
`TestSnapshotSumsQuantityAcrossSlots` and `TestSnapshotReadsEachTypeExactlyOnce`
assert exact call counts via mock-incremented counters, not just "it
worked" — a regression to per-recipe re-reads would fail
`TestSnapshotReadsEachTypeExactlyOnce`'s second assertion block.

## Independent verification

Ran from `services/atlas-maker/atlas.com/maker`:

```
$ go build ./...
$ go test ./... -count=1
ok  	atlas-maker	0.108s
ok  	atlas-maker/character	0.177s
ok  	atlas-maker/compartment	0.120s
ok  	atlas-maker/craft	0.020s
ok  	atlas-maker/crystalband	0.193s
ok  	atlas-maker/data/equipment	0.119s
ok  	atlas-maker/data/itemmake	0.172s
ok  	atlas-maker/quest	0.209s
ok  	atlas-maker/reagent	0.212s
ok  	atlas-maker/recipe	0.024s
ok  	atlas-maker/seed	0.074s
ok  	atlas-maker/skill	0.165s
$ go vet ./...        # silent
$ gofmt -l .           # silent
```

Matches the implementer's report; not taken at face value, independently
re-run.

## Non-blocking notes

- `ProcessorImpl.l` (the injected `logrus.FieldLogger`) is stored but never
  used in `craft/eligibility.go`. Consistent with existing sibling
  processors in this module (`recipe`, `compartment` — neither logs either),
  so not a new deviation introduced by this unit; noting only because it is
  otherwise dead weight in the constructor signature.
- `compartment.Builder.Build()` has no validation, unlike
  `reagent.Builder.Build() (Model, error)`. Not a defect for this unit's
  purposes; flagging so a future Builder addition to this package doesn't
  treat the no-validation shape as the established norm without deciding it
  deliberately.

## Not evaluable

- Task 23/24's actual craft-execution accommodation semantics (whether it
  checks the specific drawn reward or something else) — not yet written;
  Judgment Call A above is assessed against design/PRD text only, not
  against a consuming implementation.

## Scope confirmation

Reviewed the full diff of `06b48f3..ea58e0781`: `craft/eligibility.go`,
`craft/eligibility_test.go`, `craft/snapshot.go`, `craft/snapshot_test.go`,
`compartment/model.go`, `compartment/rest.go`. Also read, as contract
dependencies: `services/atlas-inventory/atlas.com/inventory/asset/rest.go`
and `.../compartment/processor_accommodation.go` (to verify the scope
extension's wire-field and sign-convention claims), `recipe/model.go` (to
verify the carried cache-mutation finding), `reagent/builder.go` and
`crystalband/builder.go` (to verify the Builder-pattern claim), and
`character/model.go`, `quest/model.go` (type-correctness of comparisons).
No other files in the repo were read. The commit range matches the work
found — no scope mismatch.
