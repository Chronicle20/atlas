# Review: Task 9 fix round (commits 79b0e08c5, 795f370fe)

Scope: narrow re-review of two fix commits on top of 0d239bf36, addressing the
two blocking findings of `review-task-9.md`. Files touched by these two
commits (confirmed via `git diff --name-only 0d239bf36..795f370fe`):

- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`

No other files changed in this range. Nothing under `libs/atlas-packet/`,
`libs/`, or `docs/tasks/` was touched by these commits.

Note on the prior review's error: original Blocking 1 proposed replacing
`SlotPos: 0` with `a.Slot()` from `services/atlas-channel/.../asset/model.go`.
That model is not in this consumer's import graph — the consumer imports
`atlas-channel/cashshop/inventory/asset`, whose `Model` has no `Slot()`
method. The controller ruled the wire value `0` stays and only the false
comment is corrected. This review treats that ruling as ground truth and does
not re-raise `a.Slot()`. No other derivable source is proposed here either
(none was found in the two commits' import graph).

## Blocking 1 — SlotPos comment

`consumer.go:197-210` (current, post-fix):

```
// BUY_NORMAL answers on its own SUCCESS mode byte
// (BUY_NORMAL_SUCCESS), not the generic purchase-success body --
// see cash_shop_operation.go's BUY_NORMAL arm for why the client
// gets no isPoints/currency to read on this op. Per
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md's
// BUY_NORMAL_SUCCESS row, the client reads this field as
// nPos/slotPos and passes it to
// CCSWnd_Inventory::SetSelectedNo to select which cash-shop
// inventory window entry becomes highlighted after the
// purchase -- it is not unconstrained filler. SlotPos is 0
// (selects the first entry): neither this service's nor
// atlas-cashshop's cash-locker asset model persists a
// slot/ordinal position, so 0 is an interim value pending real
// slot tracking, not a derived one.
```

Verified against `docs/tasks/task-183-cashshop-result-family/arm-catalog.md`
row for `BUY_NORMAL_SUCCESS`:

> slotPos offset 2 (u16, `movzx ebp, word ptr [esi]`@0x495394, passed as
> `nPos` to `CCSWnd_Inventory::SetSelectedNo`@0x4953d2)

The comment's citation of the offset-2 u16 field, the `nPos`/`SetSelectedNo`
sink, and the address `@0x4953d2` are all accurate to the catalog entry. The
comment no longer claims "no derivation source assigns this entry a
position" (the false claim that triggered the original finding); it now
correctly states the field IS read by the client for a real purpose
(highlighting a cash-shop window entry) while being explicit that `0` is an
interim placeholder pending real cash-locker slot tracking, not a derived
value. This matches the controller's ruling precisely.

Wire value unchanged: `SlotPos: 0,` still present at `consumer.go:214`. Wire
shape (struct literal fields, field order, types) is untouched by this diff.

**PASS.**

## Blocking 2 — BUY_NORMAL routing discrimination test

`consumer_test.go` (795f370fe) adds:
- `decodeBuyNormalDone` helper (decodes the last announced body as
  `cashpkt.BuyNormalDone`).
- `TestPurchaseSuccessBuyNormalAnnouncesBuyNormalDone`, which drives
  `handleStatusEventPurchase` with `Operation: cashshop2.ErrorOperationBuyNormal`
  and asserts: exactly one write via `cashpkt.CashShopOperationWriter`; the
  decoded mode equals `env.modeFor(cashpkt.CashShopOperationBuyNormalDone)`
  (141 for this tenant's GMS 83.1 template, per 79b0e08c5's fixture addition)
  and explicitly does NOT equal `env.modeFor(cashpkt.CashShopOperationPurchaseSuccess)`
  (66); and that `refs` contains exactly one entry with `ItemId == 5000000`,
  `Quantity == 1`.

Independent reproduction (not trusting the report's quoted output):

1. Confirmed baseline is GREEN: `go test ./kafka/consumer/cashshop/...
   ./socket/handler/...` → `ok` for both packages.
2. Mechanically deleted the `if e.Body.Operation ==
   cashshop2.ErrorOperationBuyNormal { ... }` block from
   `consumer.go:handleStatusEventPurchase` (brace-matched removal, verified
   the removed span started at `if e.Body.Operation ==
   cashshop2.ErrorOperationBuyNormal {` and ended at the block's closing
   `}`). Build still succeeds (falls through to the generic path).
3. Ran the same two-package test command with `-v`; exactly one test failed:
   `--- FAIL: TestPurchaseSuccessBuyNormalAnnouncesBuyNormalDone`. Every
   other test in both packages passed, including
   `TestPurchaseSuccessUnrelatedBuyTakesPreExistingPath` and the two
   apply-wishlist/buy-normal-failed tests in `socket/handler`.
4. Failure detail observed directly:
   `mode = 66, want the BUY_NORMAL_SUCCESS mode 141` and `mode = 66, must
   not be the generic PURCHASE_SUCCESS mode 66` — i.e. with the branch
   removed, the handler falls through to the generic PURCHASE_SUCCESS body
   (mode 66) instead of BUY_NORMAL_SUCCESS (mode 141). This is consistent
   with the report's quoted RED proof; no material disagreement observed.
5. Restored via `git checkout -- kafka/consumer/cashshop/consumer.go`;
   confirmed `git status --short` clean on that file and reran the same test
   command: GREEN (`ok` for both packages).

This reproduction independently confirms: (a) the test fails without the
fix and passes with it, (b) it isolates to the intended branch (only this
one test flips), and (c) no other test in either package is coupled to the
removed branch.

**Bidirectionality** — confirmed both directions are asserted by the suite:
- BUY_NORMAL-discriminated event → BUY_NORMAL_SUCCESS body: asserted by the
  new test (mode 141, and explicit `!=` assertion against mode 66).
- Undiscriminated event (zero-value `Operation`, i.e. `""`) → generic
  `CashShopCashInventoryPurchaseSuccessBody`: asserted by the pre-existing
  `TestPurchaseSuccessUnrelatedBuyTakesPreExistingPath`
  (`consumer_test.go:1002-1028`), which checks `body.Mode() ==
  env.modeFor(cashpkt.CashShopOperationPurchaseSuccess)`. Verified
  `cashshop2.ErrorOperationBuyNormal = "BUY_NORMAL"` (non-empty string
  constant, `kafka/message/cashshop/kafka.go:144`), so the zero-value
  `Operation` field in that pre-existing test does not accidentally satisfy
  the BUY_NORMAL branch condition — the two tests genuinely exercise
  opposite arms of the `if e.Body.Operation ==
  cashshop2.ErrorOperationBuyNormal` discriminator.

**PASS.**

## Regression checks

- Mode bytes: both the new test and the production branch resolve mode via
  `env.modeFor(...)` / the real writer-options path (per 79b0e08c5's
  fixture addition of `cashcb.CashShopOperationBuyNormalDone` — actually
  named `cashpkt.CashShopOperationBuyNormalDone` in code, see identifier
  note below); no hard-coded mode byte introduced in these two commits.
- 158/159 (or here, 66/141) success/failure pair: the failed-purchase arm
  (`cashpkt.CashShopOperationBuyNormalFailed` per
  `socket/handler/cash_shop_buy_normal_test.go` — outside this diff's
  files) is untouched by these two commits; only the success side's
  comment and test coverage changed.
- Kafka schema mirroring: no schema/struct file changed in this range (diff
  is confined to `consumer.go` and `consumer_test.go`); byte-compatibility
  claim not implicated by this diff.
- `cash_shop_credential.go` `//nolint:unused` (C3): file not touched by
  either commit (`git diff --name-only 0d239bf36..795f370fe` confirms);
  no regression possible from this diff.

**PASS** (all four, on the basis that this diff does not touch the
implicated files/values).

## Identifier resolution (import alias)

Verified directly: both `consumer.go:24` and `consumer_test.go:26` import
`cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"`.
The alias is `cashpkt`, not `cashcb` — the implementer's report statement
is correct. (79b0e08c5's own commit message text says `cashcb.CashShopOperationBuyNormalDone`,
which is a typo/inconsistency in the commit message prose only — the actual
code at `consumer_test.go` uses `cashpkt.CashShopOperationBuyNormalDone`.
Non-blocking: a cosmetic inaccuracy in a commit message, not in code.)

## Scope confirmation

`git diff --stat 0d239bf36..795f370fe` and `--name-only` both confirm the
range touches exactly the two files listed above, both under
`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/`. No
changes under `libs/atlas-packet/`, `libs/`, or `docs/tasks/`. Scope matches
the fix-only brief.

## Not evaluable

None. Both blocking findings, all four regression checks, and the identifier
question were directly verifiable within this diff's surface and its cited
dependency (arm-catalog.md).

## Verdict

Both blocking findings from the original review are genuinely closed:
Blocking 1's comment is accurate against arm-catalog.md and preserves the
controller's ruling (0 stays, comment fixed). Blocking 2's test is proven by
independent RED/GREEN reproduction, isolates to exactly the intended branch,
and asserts both directions of the discrimination. No regressions found in
the touched files; no scope creep.
