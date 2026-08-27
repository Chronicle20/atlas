# fix-ring-purchase-wrong-currency

Task: task-269-ring-pair-behavior · Branch: `task-269-ring-pair-behavior`

## Summary

Fixed BUY_COUPLE / BUY_FRIENDSHIP so the wallet currency debited on a ring
purchase reflects the player's confirmation-dialog selection (`option`) on
GMS >= 83, instead of always falling through to the prepaid bucket.

## What changed

- `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
  - Added `resolveRingPurchaseCurrency(option, isPoints, currency uint32) uint32`
    beside `resolvePurchaseCurrency`, documenting the D4a citation
    (`docs/tasks/task-240-cash-shop-stub-operations/derivation.md` §6):
    `option 1 -> 1` (NX Credit), `option 2 -> 2` (Maple Point),
    `option 4 -> 3` (NX Prepaid); any other option value (including 0)
    falls back to `resolvePurchaseCurrency(isPoints, currency)`.
  - Added an `option uint32` parameter to `RequestRingPurchase` (both the
    `Processor` interface declaration and the `ProcessorImpl` method), and
    switched its currency resolution from `resolvePurchaseCurrency` to
    `resolveRingPurchaseCurrency(option, isPoints, currency)`.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring.go`
  - Threaded `option` through `handleRingPurchase`'s signature; `handleBuyCouple`
    now passes `sp.Option()`, `handleBuyFriendship` passes `sp.Option()`.
  - Replaced the stale "option is deliberately NOT read here" doc comment on
    `handleRingPurchase` with one documenting the opposite: option IS read,
    why (GMS >= 83's ring arms carry the payment choice only in option), and
    where it flows (straight through to
    `cashshop.Processor.RequestRingPurchase`, which resolves the wallet
    currency via `resolveRingPurchaseCurrency`).
- `services/atlas-channel/atlas.com/channel/cashshop/processor_test.go`
  - Added `TestResolveRingPurchaseCurrency`, a table test covering: option 1
    (-> 1), option 2 (-> 2), option 4 (-> 3), option 0 with `isPoints=true`
    (-> 2, MaplePoints fallback), option 0 with a legacy non-zero `currency`
    (-> passthrough), and an unexpected option value (-> fallback to
    `resolvePurchaseCurrency`, which with the given inputs is 0).

## Not changed (scope)

- `handleBuyPackage` / `cash_shop_package.go` (BUY_PACKAGE, mode 30) has the
  identical defect but is explicitly out of scope per this task's ruling —
  separately tracked.
- No packet/codec change: `Option()` was already decoded and exposed on both
  `ShopOperationBuyCouple` and `ShopOperationBuyFriendship` (confirmed via
  `go doc github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound.ShopOperationBuyCouple` /
  `...ShopOperationBuyFriendship`).
- No change in `atlas-cashshop`: once currency 1/2/3 arrives on the command,
  `processor_ring.go` and `wallet.Model.Balance` already behave correctly.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring_test.go`
  was not extended: `handleRingPurchase` constructs `character.NewProcessor`
  and `cashshop.NewProcessor` directly (no injected mock/producer seam in
  that test file) and the brief explicitly forbids inventing a
  test-only constructor. The option-passthrough is instead covered at the
  processor layer (`resolveRingPurchaseCurrency`) and confirmed by reading
  the diff: `handleBuyCouple`/`handleBuyFriendship` pass `sp.Option()`
  straight through to `handleRingPurchase`, which passes it straight through
  to `RequestRingPurchase`.
- JMS ring arms and BUY_PACKAGE remain flagged, not fixed, per the brief's
  "Not yet answered" section — unchanged by this fix.

## Testing

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```
No output (success).

```
cd services/atlas-channel/atlas.com/channel && go test ./...
```
All packages `ok` (pristine, no failures, no stray warnings).

```
cd services/atlas-channel/atlas.com/channel && go test ./cashshop/... -run TestResolveRingPurchaseCurrency -v
```
```
=== RUN   TestResolveRingPurchaseCurrency
=== RUN   TestResolveRingPurchaseCurrency/option_1_->_NX_Credit_(wallet_1)
=== RUN   TestResolveRingPurchaseCurrency/option_2_->_Maple_Point_(wallet_2)
=== RUN   TestResolveRingPurchaseCurrency/option_4_->_NX_Prepaid_(wallet_3)
=== RUN   TestResolveRingPurchaseCurrency/option_0,_isPoints_true_->_falls_back_to_MaplePoints_(2)
=== RUN   TestResolveRingPurchaseCurrency/option_0,_legacy_non-zero_currency_->_passthrough
=== RUN   TestResolveRingPurchaseCurrency/unexpected_option_value_->_falls_back_to_resolvePurchaseCurrency
--- PASS: TestResolveRingPurchaseCurrency (0.00s)
    --- PASS: TestResolveRingPurchaseCurrency/option_1_->_NX_Credit_(wallet_1) (0.00s)
    --- PASS: TestResolveRingPurchaseCurrency/option_2_->_Maple_Point_(wallet_2) (0.00s)
    --- PASS: TestResolveRingPurchaseCurrency/option_4_->_NX_Prepaid_(wallet_3) (0.00s)
    --- PASS: TestResolveRingPurchaseCurrency/option_0,_isPoints_true_->_falls_back_to_MaplePoints_(2) (0.00s)
    --- PASS: TestResolveRingPurchaseCurrency/option_0,_legacy_non-zero_currency_->_passthrough (0.00s)
    --- PASS: TestResolveRingPurchaseCurrency/unexpected_option_value_->_falls_back_to_resolvePurchaseCurrency (0.00s)
PASS
ok  	atlas-channel/cashshop	0.009s
```

## Self-review

- Signature change to `RequestRingPurchase` on both the interface and the
  impl; grepped the whole `atlas-channel` module for other call sites —
  only the handler calls it, no mock caller to update.
- `option` is threaded through as a plain positional `uint32`, matching the
  existing style of `isPoints`/`currency` in the same function signatures
  (no wrapper struct introduced — YAGNI, and the brief didn't ask for one).
- The stale doc comment claiming option is "deliberately NOT read" was fully
  replaced, not just amended around — it now describes the new behavior and
  the D4a evidence for why it changed, and cites the bug file.
- Did not touch `cash_shop_package.go` / `handleBuyPackage` per the explicit
  ruling.
- Did not touch any codec in `libs/atlas-packet`.

## Concerns

None. Module-local build and tests are clean.

## Commit

Committed on `task-269-ring-pair-behavior` (see final message for SHA).
