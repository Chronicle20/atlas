# bug-ring-purchase-wrong-currency

Task: task-269-ring-pair-behavior · Branch: `task-269-ring-pair-behavior` · PR env: `atlas-pr-1524`

## Reproduced

Yes. Tenant is **GMS 83.1** (confirmed from the tenant config message in
`atlas-cashshop` pod logs, namespace `atlas-pr-1524`:
`"region":"GMS","majorVersion":83,"minorVersion":1`).

Buy a friendship ring from the cash shop with an account holding ~100,000 NX
Credit and 0 NX Prepaid, selecting **NX Credit** in the confirmation dialog.

## Observed

The client shows a "not enough cash" rejection. `atlas-cashshop` logs
(`atlas-cashshop-644cc88798-tmtsh`, `--tail=2000`):

```
insufficient balance for ring purchase. Cost [3600]. Balance [0].
insufficient balance for ring purchase. Cost [3600]. Balance [0].
```

The reported balance is 0 while the account's NX Credit balance is ~100,000 —
the check read the **prepaid** bucket, not credit.

## Expected

The purchase debits the currency the user selected in the confirmation dialog
(NX Credit → `wallet.Model` currency 1), and succeeds when that bucket covers
the price.

## Root cause

On GMS ≥ 83, `CCashShop::OnBuyFriendship` / `OnBuyCouple` do **not** put
`isPoints`/`currency` on the wire. The v83+/v95 decode path in
`libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:118-127`
(and the identical `shop_operation_buy_couple.go:109-127`) reads
`birthday|spw, option, serialNumber, name, message` — `isPoints` stays `false`
and `currency` stays `0`.

The payment selection lives entirely in **`option`**. That is established
IDA-derived evidence, not inference:
`docs/tasks/task-240-cash-shop-stub-operations/derivation.md` §6 (D4a) —
`CCashShop::OnBuyCouple` @ `0x490d80` seeds `dwOption` with an affordability
bitmask and `CConfirmPurchaseDlg::Confirm` @ `0x48c100` writes the user's radio
selection back through the pointer, constrained to exactly one of
**1 = NX Credit, 2 = Maple Point, 4 = NX Prepaid**. §6 Step 3 states explicitly
that `pointType` alone cannot distinguish 1 from 4, and that "any future
implementation that debits a real balance must read `option`."

`handleRingPurchase` deliberately does not read `option`
(`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring.go:93-102`),
so it forwards `isPoints=false, currency=0`.
`resolvePurchaseCurrency` (`services/atlas-channel/atlas.com/channel/cashshop/processor.go:132-138`)
only rewrites `currency` when `isPoints && currency == 0`, so `0` passes
through unchanged onto the Kafka `RequestRingPurchaseCommandBody.Currency`.

On the consumer side, `w.Balance(currency)`
(`services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-45`) treats
anything that is not 1 or 2 as **prepaid**, so `currency == 0` reads the
prepaid balance (0) and
`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_ring.go:159-162`
rejects with `NOT_ENOUGH_CASH`.

**BUY_COUPLE has the identical defect** — it shares `handleRingPurchase` and
its v83+ decode path is the same shape. Both arms must be fixed together.

## Fix

Resolve the wallet currency from `option` on the ring arms, falling back to the
existing `isPoints`/`currency` resolution when `option == 0` (legacy GMS < 83
carries a real `currency` int; JMS carries neither and keeps today's prepaid
behavior — see Not yet answered).

Mapping, per D4a and the `wallet.Model.Balance` convention:
`option 1 → 1` (credit), `option 2 → 2` (points), `option 4 → 3` (prepaid),
anything else → fall back to `resolvePurchaseCurrency(isPoints, currency)`.

Keep the resolution in the `cashshop` processor next to
`resolvePurchaseCurrency` — one place owns currency resolution — rather than
duplicating a bitmask mapper in the handler.

File inventory:

- `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
  — add `resolveRingPurchaseCurrency(option, isPoints, currency uint32) uint32`
    beside `resolvePurchaseCurrency` (line 127-138), documenting the D4a
    citation; add an `option uint32` parameter to `RequestRingPurchase`
    (line 307-311) and use the new resolver in place of
    `resolvePurchaseCurrency`.
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go` interface
  declaration for `RequestRingPurchase` (same file, `Processor` interface) —
  signature change.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring.go`
  — thread `option` through `handleRingPurchase`; `handleBuyCouple` passes
    `sp.Option()`, `handleBuyFriendship` passes `sp.Option()`. **Replace** the
    "option is deliberately NOT read here" paragraph in the `handleRingPurchase`
    doc comment (lines 93-102) — it now documents the opposite behavior and
    must not be left stale.
- `services/atlas-channel/atlas.com/channel/cashshop/` — add a table test for
  `resolveRingPurchaseCurrency` covering option 1/2/4, option 0 with
  `isPoints=true` (→ 2), option 0 with a legacy non-zero `currency`
  (→ passthrough), and an unexpected option value (→ fallback).
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring_test.go`
  — extend if the existing mock/producer seam makes the option passthrough
    assertable; do not invent a test-only constructor (repo rule).

No change is needed in `atlas-cashshop`: once a correct currency (1/2/3)
arrives on the command, `processor_ring.go` and `wallet.Model` already behave.

No packet/codec change: `option` is already decoded and exposed via
`ShopOperationBuyCouple.Option()` / `ShopOperationBuyFriendship.Option()`.

## Not yet answered

- **JMS ring arms** carry no payment signal at all (`decodeJMS` reads
  `spw, serialNumber, name, message`), so they keep resolving to prepaid. No
  evidence pins JMS's intended ring currency; unchanged by this fix, flagged
  not fixed.
- **BUY_PACKAGE (mode 30) has the same defect** —
  `handleBuyPackage` (`cash_shop_package.go:84-90`) passes a literal `0`
  currency and ignores `sp.Option()` for exactly the same documented reason.
  It is outside task-269's ring surface (it belongs to task-240's already-merged
  work) and is NOT fixed here. Needs its own task.
- Whether `option` can legitimately arrive as `0` on GMS v83 (D4a §6 Step 2
  says never on a successful confirm) — the fallback path is defensive, not
  observed.

## Resolution

Fixed by `67d31d6bc` — "fix(atlas-channel): resolve ring purchase currency from
option on GMS>=83" (docs follow-up `8ee623194`). Implementation report:
`fix-ring-purchase-wrong-currency.md`.

`cash_shop_ring_test.go` was NOT extended: `handleRingPurchase` has no injected
mock/producer seam and a test-only constructor is forbidden by repo convention.
Option passthrough is covered at the processor layer by
`TestResolveRingPurchaseCurrency` instead.

The resolver introduced here was renamed by the follow-up BUY_PACKAGE fix
(`d228fea7a`) when a second arm started sharing it:
`resolveRingPurchaseCurrency` → **`resolveOptionCurrency`**. Mapping and
behavior unchanged.

Gate: flagless `tools/verify.sh --base 94a3c365c` exits 0 — "All checks
passed", including `go build/vet/test -race` on atlas-channel.

Review: `review-currency-fixes.md`, verdict APPROVED_WITH_FINDINGS, 0 blocking.

**Live re-test on `atlas-pr-1524`: NOT yet performed** — this bug is not
confirmed fixed against the client until an NX Credit friendship-ring purchase
succeeds there.
