# bug-cash-shop-live-testing-round-2 — implementation report (Defects D and E)

Scope: only Defect D and Defect E from the `## Fix` section of
`bug-cash-shop-live-testing-round-2.md`. Defects F and G and the
"Unresolved symptom" section were explicitly out of scope and were not
touched, stubbed, or annotated with TODOs.

## Defect D — by-type inventory-slot purchase now grants 4 slots

Per the ruling: the amount becomes `4` (not `8`); the `4000` cost literal is
left unchanged.

### Change
`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`,
`PurchaseInventoryIncreaseByTypeAndEmit`: changed the hard-coded amount
argument passed to `PurchaseInventoryIncrease` from `8` to `4`, matching
`PurchaseInventoryIncreaseByItemAndEmit`'s amount.

### Test
Added `TestPurchaseInventoryIncreaseByTypeGrantsFourSlots` to
`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_inventoryincrease_test.go`,
following the same fixture pattern as the existing by-item tests in that
file (`purchaseTestDatabase`, `startPurchaseCharacterServer`,
`seedPurchaseWallet`). It calls `PurchaseInventoryIncreaseByTypeAndEmit`
with `inventory.TypeValueUse` (matching the live repro's inventory type 2),
asserts:
- the wallet is debited exactly 4000 (the unchanged cost),
- the emitted `InventoryCapacityIncreasedBody.Amount == 4`,
- `Capacity == 4` (the stubbed character's compartment lookup returns an
  empty/zero-capacity compartment in this harness, same as the existing
  by-item tests' documented behavior, so newCapacity == 0 + 4).

### Docs
`services/atlas-cashshop/docs/domain.md` had two "8 slots" mentions of the
by-type grant (lines 408 and 415); both updated to "4 slots" to match the
new behavior.
`services/atlas-cashshop/docs/kafka.md` was checked too
(`grep -rn "8 slots" services/atlas-cashshop/`); its two `"amount": 8`
occurrences are generic example JSON for `IncreaseCapacityCommandBody` and
`InventoryCapacityIncreasedBody` — not textually "8 slots" and not
specifically labeled as the by-type example — so they were left alone per
the brief's literal grep target.

## Defect E — buy-for-self package purchase now announces BUY_PACKAGE_SUCCESS

Fixed on the consumer side, per the ruling; the producer and the Kafka body
shape are unchanged.

### Change
`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`,
`handleStatusEventPackagePurchased`:
- Replaced the discriminator `e.Body.RecipientCharacterId != 0` with
  `e.Body.RecipientCharacterId != e.CharacterId`, since the STATUS body
  (unlike the COMMAND body) always echoes a concrete recipient identity —
  the buyer's own on a buy-for-self purchase — per
  `kafka/message/cashshop/kafka.go:372-375`.
- Rewrote the function's doc comment (previously asserting the ZERO
  convention) to state the status body's actual convention and cite the
  kafka.go doc comment location.
- Confirmed (via `grep -n "RecipientCharacterId"
  services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`)
  that this is the only `RecipientCharacterId` discriminator in the file —
  no other handler copied the `!= 0` convention.

### Test
`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`:
- `TestPackagePurchasedBuyForSelfProjectsAssetsIntoBuyPackageDone`: changed
  the fixture's `RecipientCharacterId` from `0` to `testCharacterId` (the
  buyer's own id, matching the real status body's convention) and
  `RecipientName` from `""` to `"Buyer"`. Updated the doc comment to
  explain why. It already asserts the announced arm decodes as
  `BuyPackageDone` with `len(body.Items()) == 1`, matching
  `len(Body.AssetIds) == 1` in the fixture.
- `TestPackagePurchasedGiftAnnouncesGiftPackageDone`: left the fixture as-is
  (`RecipientCharacterId: 99`, which already differs from
  `testCharacterId == 7`); updated its doc comment to describe the new
  discriminator (`!= CharacterId` instead of `!= 0`).

## Build / test evidence

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
All packages `ok` (or `[no test files]`), including
`atlas-cashshop/cashshop` (contains the new by-type test).

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages `ok` (or `[no test files]`), including
`atlas-channel/kafka/consumer/cashshop` (contains the updated
package-purchased tests).

Both build with no errors, no vet output, and all tests pass.

## Formatting

Ran `tools/lint.sh` (no flags = fix mode, the repo's formatting authority)
tree-wide before committing. Output: `0 issues.` repeated for every module,
exit code 0. No files were rewritten by the formatter (`git status --short`
showed no additional diffs beyond the manual edits), so no separate
formatting commit was needed.

## Files changed

Commit 1 (`648310b18`) — Defect D:
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_inventoryincrease_test.go`
- `services/atlas-cashshop/docs/domain.md`

Commit 2 (`59916a651`) — Defect E:
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`

## Self-review

- Confirmed the `4000` cost literal for Defect D is untouched — only the
  amount argument changed.
- Confirmed the producer (`processor_package.go`) and the Kafka body shape
  (`kafka.go`'s `PackagePurchasedBody`) for Defect E are untouched — only
  the consumer's discriminator and doc comment changed, per the ruling.
- Confirmed no stray `!= 0` copies of the old discriminator remain in
  `consumer.go`.
- Confirmed Defects F, G, and the unresolved-symptom section were not
  touched: no code changes outside the two files above (plus their test
  files and the one docs file), no new TODOs, no stubs.
- Verified branch/worktree after both commits: `git rev-parse
  --show-toplevel` and `git branch --show-current` both correct
  (`task-240-cash-shop-stub-operations` worktree/branch). `git status
  --short` shows only the pre-existing untracked brief file
  (`docs/tasks/task-240-cash-shop-stub-operations/bug-cash-shop-live-testing-round-2.md`),
  which was already untracked before this task started and is not part of
  this deliverable.

## Issues or concerns

None. Both fixes are narrowly scoped, module-local tests pass, and the
brief's exact wording (amount `4`, cost `4000` unchanged; consumer-side fix
for Defect E, producer/body shape unchanged) was followed verbatim.
