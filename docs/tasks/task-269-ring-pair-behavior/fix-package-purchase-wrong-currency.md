# fix-package-purchase-wrong-currency

Task: task-269-ring-pair-behavior · Branch: `task-269-ring-pair-behavior`
Commit: `d228fea7a`

## What I implemented

Per `docs/tasks/task-269-ring-pair-behavior/bug-package-purchase-wrong-currency.md`:

1. **Renamed the shared resolver.** `resolveRingPurchaseCurrency(option, isPoints, currency)`
   in `services/atlas-channel/atlas.com/channel/cashshop/processor.go` is now
   `resolveOptionCurrency(option, isPoints, currency)` — an arm-neutral name,
   since the option→wallet-currency mapping (1=NX Credit, 2=Maple Point,
   4=NX Prepaid → wallet codes 1/2/3) is a property of `dwOption` itself, not
   of the ring arms. Updated its doc comment to describe both the ring arms
   and BUY_PACKAGE as callers.
2. **`RequestPackagePurchase`** (interface declaration + `ProcessorImpl`
   method) gained a new `option uint32` parameter, inserted after `currency`
   to match `RequestRingPurchase`'s existing parameter order. It now resolves
   through `resolveOptionCurrency(option, isPoints, currency)` instead of the
   plain `resolvePurchaseCurrency(isPoints, currency)`. Rewrote the doc
   comment at lines ~309-329 that previously called `resolvePurchaseCurrency`
   "load-bearing" for BUY_PACKAGE and explained why BUY_OTHER_PACKAGE's call
   was an inert passthrough under that helper — it now explains the same
   inertness under `resolveOptionCurrency` with `option=0`.
3. **`RequestRingPurchase`**'s call site updated to call `resolveOptionCurrency`
   (same arguments, same behavior) and its doc comment's reference to
   `resolveRingPurchaseCurrency` updated to the new name.
4. **`handleBuyPackage`** (`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package.go`)
   now passes `sp.Option()` instead of a literal `0` as the new `option`
   argument. Replaced the doc comment paragraph that explained why `option`
   was deliberately not read — it now explains that `option` carries the
   confirmation dialog's actual payment-method choice and is resolved via
   `resolveOptionCurrency`, mirroring the ring arms.
5. **`handleBuyOtherPackage`**'s call site updated to pass a literal `0` for
   the new `option` parameter (gift package stays pinned to
   `walletCurrencyPrepaid`, per the brief's explicit ruling). Its doc comment
   reference to `resolvePurchaseCurrency` updated to `resolveOptionCurrency`
   to reflect the renamed helper, still describing the same inert-passthrough
   behavior.
6. **`cash_shop_ring.go`**'s `handleRingPurchase` doc comment previously said
   option "IS read here, unlike handleBuyPackage's D4a treatment" and cited
   `resolveRingPurchaseCurrency` — both are now stale (handleBuyPackage now
   also reads option; the resolver is renamed). Rewrote to say handleBuyPackage
   "now applies" the same D4a treatment, and updated the resolver name.
7. **Test:** renamed `TestResolveRingPurchaseCurrency` →
   `TestResolveOptionCurrency` in `processor_test.go`, updated its doc comment
   to describe both callers, and updated the call/assertion inside to
   `resolveOptionCurrency`. No new test case was needed — BUY_PACKAGE uses the
   identical option value space (1/2/4/0-fallback) already covered by the
   existing table.

## Out of scope (per brief ruling, unchanged)

- `handleBuyOtherPackage` (mode 31, gift package) still hardcodes
  `walletCurrencyPrepaid`; it only gained the new `option=0` argument to the
  call site.
- No packet/codec change — `Option()` was already decoded and exposed by
  `ShopOperationBuyPackage`.
- No `atlas-cashshop` change.

## Tested

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```
→ exit 0, no output.

```
cd services/atlas-channel/atlas.com/channel && go test ./cashshop/... ./socket/handler/...
```
→
```
ok  	atlas-channel/cashshop	0.008s
ok  	atlas-channel/cashshop/inventory	(cached)
ok  	atlas-channel/cashshop/inventory/asset	(cached)
ok  	atlas-channel/cashshop/inventory/compartment	(cached)
ok  	atlas-channel/cashshop/item	(cached)
ok  	atlas-channel/cashshop/purchaserecord	(cached)
?   	atlas-channel/cashshop/wallet	[no test files]
ok  	atlas-channel/cashshop/wishlist	(cached)
ok  	atlas-channel/socket/handler	2.229s
```

Then the full module suite:

```
cd services/atlas-channel/atlas.com/channel && go test ./...
```
→ all packages `ok` or `[no test files]`, no failures.

## Files changed

- `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
- `services/atlas-channel/atlas.com/channel/cashshop/processor_test.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring.go`

## Self-review

- Confirmed no other call sites of `RequestPackagePurchase` or
  `resolveRingPurchaseCurrency` remained unpatched (`grep -rn` across
  `services/atlas-channel/atlas.com/channel`).
- Confirmed `handleBuyOtherPackage` behavior is unchanged: still passes
  `false, walletCurrencyPrepaid`; only the new trailing `option` argument
  (`0`) was added, which resolves through the fallback branch to the same
  passthrough as before.
- Confirmed no packet/codec files under `libs/atlas-packet` were touched.
- No stale doc comments referencing the old resolver name or the old
  "option deliberately not read" rationale remain in the four touched files.

## Issues or concerns

None. `option` argument insertion point in `RequestPackagePurchase` was
placed to mirror `RequestRingPurchase`'s existing `(isPoints, currency,
option, serialNumber, ...)` ordering for consistency across the two BUY
arms that now share the resolver.
