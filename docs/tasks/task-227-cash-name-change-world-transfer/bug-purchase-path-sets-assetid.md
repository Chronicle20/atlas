# Bug: the cash-shop purchase path sets `assetId`, so cancel refunds a coupon that was never consumed

Live evidence from `atlas-pr-1370`, tenant `d606f1cb-ba79-45ca-a989-cf0dc956fee7`,
character 1 (`Atlas2`), 2026-08-16 15:29 UTC. All quotes are verbatim pod-log lines.

## Answer to the reported question

**Yes — the name change really was cancelled, exactly once.** There was no
double-cancel and no redelivered resolve.

```
15:29:17.710 atlas-character  Created pending change [922af6c9-9769-41f3-b44a-744269513830]
                              type [NAME_CHANGE] for character [1] in world [0].
15:29:48.531 atlas-character  Handling [POST] request on [/api/characters/1/pending-changes/cancel]
15:29:48.541 atlas-character  Pending change [922af6c9-...] for character [1] transitioned
                              PENDING -> CANCELLED, reason [player_cancelled].
```

One create, one cancel, one `player_cancelled`. The record is terminal. The
`Resolve` idempotency guard (`processor.go` — emit only when `moved == true`)
was never exercised twice; the second coupon is not a redelivery.

## What actually produced the second coupon

The two awards have different provenance. Neither is a duplicate of the other —
the first is the purchase, the second is a refund for a consumption that never
happened.

| Time | What | Result |
|---|---|---|
| 15:29:17.710 | pending change created **with `assetId = 5400000`** | record `922af6c9` |
| 15:29:17.764 | `destroy_asset` saga `e9be7f6d` (`consume_pending_change_coupon`, templateId 5400000) | emitted because `HasAsset()` was true |
| 15:29:17.778 | saga GETs `/api/characters/1/inventory/compartments?type=5` → **`Unable to destroy asset.`** / `Saga step handler returned a synchronous error.` / `Failed to insert saga into cache` | **consumption FAILED, silently, no compensation** |
| 15:29:17.821 | atlas-cashshop: `Character [1] successfully purchased item [5400000] for [10000] currency` (serial 50600000, prepaid 100000 → 90000) | coupon lands in cash locker `424ba10c`, assetId 1 |
| 15:29:27 | player withdraws it (`withdraw_from_cash_shop` saga `41131ef2`) | **coupon #1** → inventory asset 27, slot 1 |
| 15:29:48.541 | cancel → `PENDING -> CANCELLED` | — |
| 15:29:48.595 | `award_asset` saga `22181864` (`refund_pending_change_coupon`, templateId 5400000), emitted because `HasAsset()` is still true | **coupon #2** → inventory asset 28, slot 2 |

Net state: 2 × item 5400000 held, 10 000 NX charged, name unchanged.

## Root cause

`atlas-character` documents the invariant in two places:

- `pending_change/entity.go:69-71` — *"AssetId is null on the cash-shop purchase
  path, which carries an entitlement reference correlated by TransactionId
  instead of an asset."*
- `pending_change/processor.go:255-262` — *"The purchase path has no asset to
  destroy here."*

`atlas-channel` violates it. Both cash-shop purchase handlers pass the
commodity's template id as the asset id:

- `socket/handler/cash_shop_operation.go:264` — `RequestNameChange(characterId, sp.NewName(), com.ItemId)`
- `socket/handler/cash_shop_operation.go:311` — `RequestWorldTransfer(characterId, destinationWorldId, com.ItemId)`

and the processor signatures make sending null impossible — `assetId uint32`,
not `*uint32` (`channel/pendingchange/processor.go:31-32,61,70`).

**These are the only two callers in the repo.** Grep for
`RequestNameChange|RequestWorldTransfer` over all non-test `.go` files returns
nothing else. So the "item path" that `assetId` exists to serve has no producer
at all today — every pending change ever created goes through the purchase path,
and every one of them sets `assetId` wrongly. `character_cash_item_use.go` only
wires the *cancel* arm (`handleCashCouponCancel`, lines 771/775), never a
request.

Consequences, both live on every request:

1. A `destroy_asset` saga fires for a coupon the player does not hold at that
   instant, fails synchronously, and nobody notices — the failure is logged at
   `debug` and never surfaces to the client or compensates the record.
2. On any non-`APPLIED` exit (cancel, expiry, rejection) the `Resolve` refund
   branch mints a coupon that was never consumed. The idempotency guard is
   working correctly and is not the problem — the guard protects against a second
   refund, not against a first refund that shouldn't exist.

The world-transfer arm has the identical shape, so the same duplication is
expected there.

## Contradiction to resolve while fixing

`pending_change/producer.go:82-84` says the purchase path's entitlement *"is
consumed by atlas-cashshop off PENDING_CHANGE_CREATED"*, while
`pending_change/processor.go:258-262` says atlas-cashshop *"does not consume"*
that event. The live logs agree with the second: atlas-cashshop's only
involvement is the `REQUEST_PURCHASE` round trip on `COMMAND_TOPIC_CASH_SHOP`
(15:29:17.762, transactionId = the pending-change id). Whatever the fix, one of
those two comments is wrong and should go.

Also worth deciding: whether a direct NX name-change purchase should award a
coupon into the cash locker at all. atlas-cashshop's generic commodity flow does
it unconditionally (`successfully purchased item [5400000]`), which is why the
player ends up holding a coupon for a change they paid NX for directly.

## Fix

`atlas-channel` no longer sends an assetId on this path, and cannot:
`RequestNameChange` / `RequestWorldTransfer` lost the parameter, and
`CreateInputRestModel.AssetId` was removed rather than kept and set to nil —
this service has no item path, so the field only ever existed to carry the bug.
Both purchase handlers still resolve the commodity, but now solely as a
pre-insert guard against an unusable serial number (dropping the lookup would
have let a bad serial strand a PENDING record, since the record is inserted
before the purchase is requested).

Two tests pin the seam from both sides:

- `channel/pendingchange/request_body_test.go` — captures the actual POST body
  and fails if `assetId` appears in it. Verified to fail against the unfixed
  code with `"assetId":5400000` in both the name-change and world-transfer
  bodies.
- `character/pending_change/purchase_path_test.go` — the receiving half. Every
  pre-existing test in that package created with a **non-nil** assetId, so
  nothing pinned the purchase path at all: this one asserts a nil-asset record
  emits no `destroy_asset` on create and, the discriminating part, no
  `award_asset` on cancel.

`producer.go`'s claim that atlas-cashshop consumes the purchase-path entitlement
off `PENDING_CHANGE_CREATED` was removed — it contradicted `processor.go` a few
lines away, and `grep -rn PENDING_CHANGE services/atlas-cashshop/` returns
nothing.

## Left open deliberately

The `destroy_asset` saga failing synchronously (`Unable to destroy asset`) is
logged at debug, compensates nothing, and leaves the record PENDING. This fix
makes that unreachable on the only live path, but the general hole remains and
the right behaviour is a real design question — cancel the record, retry, or
surface to the player? Not guessed at here.
