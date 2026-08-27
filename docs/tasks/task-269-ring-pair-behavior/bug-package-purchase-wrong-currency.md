# bug-package-purchase-wrong-currency

Task: task-269-ring-pair-behavior · Branch: `task-269-ring-pair-behavior` · PR env: `atlas-pr-1524`

Companion to `bug-ring-purchase-wrong-currency.md` — the same defect class on
the BUY_PACKAGE arm. Fixed on the user's explicit request after the ring fix
landed, even though BUY_PACKAGE originates in task-240's merged work.

## Reproduced

Not reproduced live. It is the identical code path as the ring bug, which WAS
reproduced live on GMS 83.1 in `atlas-pr-1524`
(`insufficient balance for ring purchase. Cost [3600]. Balance [0]` with a
~100,000 NX Credit account). The BUY_PACKAGE defect is established by reading
the same three files, not by inference about behavior:

- `ShopOperationBuyPackage` decodes `pointType`, `option`, `serialNumber` on
  GMS ≥ 83 (`libs/atlas-packet/cash/serverbound/shop_operation_buy_package.go:60-70`).
  There is **no `currency` int on the wire** at all, on any version.
- `handleBuyPackage` passes `sp.PointType()` as `isPoints` and a **literal `0`**
  as `currency`, explicitly declining to read `sp.Option()`
  (`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package.go:68-90`).
- `resolvePurchaseCurrency(isPoints, currency)`
  (`services/atlas-channel/atlas.com/channel/cashshop/processor.go`) rewrites
  `0` → `2` (Maple Points) only when `isPoints` is true; otherwise `0` passes
  through, and `wallet.Model.Balance`
  (`services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-45`) treats
  anything that is not 1 or 2 as **prepaid**.

## Observed

A cash-package purchase paid with **NX Credit** (`option == 1`) debits/checks
the **prepaid** bucket, so an account with NX Credit but no prepaid balance is
rejected for insufficient funds. An **NX Prepaid** purchase (`option == 4`)
happens to work by accident, and a Maple Point purchase (`option == 2`) works
via the `pointType` bool.

## Expected

BUY_PACKAGE debits the currency the user selected in the confirmation dialog:
NX Credit → wallet currency 1, Maple Point → 2, NX Prepaid → 3.

## Root cause

Same as the ring bug. Per `docs/tasks/task-240-cash-shop-stub-operations/derivation.md`
§6 (D4a), IDA-derived: `CCashShop::OnBuyPackage` @ `0x48ed40` seeds `dwOption`
with an affordability bitmask and `CConfirmPurchaseDlg::Confirm` @ `0x48c100`
writes the user's radio selection back through the pointer, constrained to
exactly one of **1 = NX Credit, 2 = Maple Point, 4 = NX Prepaid**. Step 3 of
that section states that `pointType` is `dwOption == 2` and is therefore lossy:
a server reading only `pointType` **cannot distinguish NX Credit (1) from NX
Prepaid (4)** — both arrive as `false` and collapse onto prepaid.

`handleBuyPackage` reads only `pointType`.

## Fix

Read `option` on the BUY_PACKAGE arm and resolve the wallet currency from it,
exactly as the ring fix did.

The ring fix (`bug-ring-purchase-wrong-currency.md`, already committed on this
branch) introduced the option→wallet-currency resolver in
`services/atlas-channel/atlas.com/channel/cashshop/processor.go`. **Reuse that
resolver — do not write a second mapper.** If its name is ring-specific
(`resolveRingPurchaseCurrency` — it was renamed to `resolveOptionCurrency`), rename it to an arm-neutral name and update
the ring call site and its tests in the same commit; the mapping is a property
of `dwOption`, not of the ring arms.

File inventory:

- `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
  — add an `option uint32` parameter to `RequestPackagePurchase` (line ~325)
    and resolve through the shared option resolver instead of
    `resolvePurchaseCurrency`. Update the `Processor` interface declaration.
    Update the doc comment at lines ~307-324, which currently states that
    `resolvePurchaseCurrency` is "load-bearing" for BUY_PACKAGE — that becomes
    stale and must not be left.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package.go`
  — `handleBuyPackage` passes `sp.Option()`. **Replace** the paragraph in its
    doc comment (lines ~68-83) explaining why `option` is deliberately not read;
    it now documents the opposite behavior.
- `services/atlas-channel/atlas.com/channel/cashshop/` tests — extend the
  shared resolver's table test (added by the ring fix) if the package arm adds
  a case it does not already cover.

**Explicitly OUT of scope — do not change `handleBuyOtherPackage`** (mode 31,
gift package). Per derivation.md D3a, `CCashShop::OnGiftPackage`'s body has no
`pointType`/`option` field at all and the arm is pinned to NX Prepaid
(`dwOption = 4` is never encoded); it passes the hardcoded
`walletCurrencyPrepaid` constant on purpose. That call site keeps its current
behavior — pass `0` for the new `option` parameter so the fallback preserves
the existing already-final currency passthrough.

No change in `atlas-cashshop`: a correct currency (1/2/3) on the command is
already handled. No packet/codec change: `option` is already decoded and
exposed via `ShopOperationBuyPackage.Option()`.

## Not yet answered

- **Legacy GMS < 83** decodes only `serialNumber` for BUY_PACKAGE
  (`shop_operation_buy_package.go:64-66`), so it carries neither `pointType`
  nor `option` and continues to resolve to prepaid. No evidence pins what v79
  intended; unchanged by this fix, flagged not fixed.
- Whether `option` can legitimately arrive as `0` on GMS v83 (D4a §6 Step 2
  says never on a successful confirm) — the fallback path is defensive, not
  observed.

## Resolution

Fixed by `d228fea7a` — "fix(atlas-channel): resolve BUY_PACKAGE currency from
confirmation-dialog option". Implementation report:
`fix-package-purchase-wrong-currency.md`.

The shared resolver this brief asked to reuse was renamed as instructed:
`resolveRingPurchaseCurrency` → **`resolveOptionCurrency`**, with
`TestResolveRingPurchaseCurrency` → `TestResolveOptionCurrency`. Any earlier
mention of the ring-specific name in this file or in
`fix-ring-purchase-wrong-currency.md` refers to that same resolver.

Gate: flagless `tools/verify.sh --base 94a3c365c` exits 0 — "All checks
passed", including `go build/vet/test -race` on atlas-channel.

Review: `review-currency-fixes.md`, verdict APPROVED_WITH_FINDINGS, 0 blocking.
The reviewer independently confirmed the load-bearing seam property — option 4
maps to wallet currency **3**, never a bare 4 — against `wallet.Model` and
`effectivePurchaseCurrency` on the atlas-cashshop side, and confirmed
`handleBuyOtherPackage` resolves to 3 exactly as before. Both non-blocking
findings were documentation-only and are closed by this commit.

**Live re-test on `atlas-pr-1524`: NOT yet performed** — an NX Credit cash
package purchase has not been exercised against the client.
