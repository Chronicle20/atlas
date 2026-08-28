# Defect F — implementation brief: surface the gift sender and message to the recipient

Companion to `bug-cash-shop-live-testing-round-2.md` § "Defect F". That file has
the symptom and the diagnosis; this file has the design ruling and the file
inventory. Read both. Branch `task-240-cash-shop-stub-operations`, worktree
`.worktrees/task-240-cash-shop-stub-operations`.

## The design question, and how the client answered it

The open worry when F was triaged was whether gifts must sit in a *pending*
list until the recipient accepts them (our implementation delivers the asset
straight into the recipient's locker). If the client had an "accept gift"
request, direct delivery would double-count.

**It does not.** The v83 export (`docs/packets/ida-exports/gms_v83.json`) lists
every serverbound `CCashShop::*` function, and the only gift-related one is
`CCashShop::SendGiftsPacket @ 0x46f940` — which is the *send a gift* packet
(mode 4: sub-action byte, field A, serialNumber, recipient name, message),
already handled at `services/atlas-channel/.../socket/handler/cash_shop_gift.go`.
There is no gift-list request and no accept-gift request.

`LOAD_GIFT_SUCCESS` is therefore a **server-pushed, display-only** list, exactly
like `LOAD_INVENTORY_SUCCESS` and `LOAD_WISHLIST` in the cash-shop entry burst.
Direct delivery is compatible with it. No user ruling is needed; implement it as
described below.

## Root cause chain (all three links are already established — do not re-derive)

1. `GiftFrom`/`GiftMessage` **are** persisted, on both entity and model
   (`services/atlas-cashshop/.../inventory/asset/entity.go:48-57`,
   `.../asset/model.go:77-88`).
2. They are **not exposed over REST**: `asset.RestModel` and `Transform`
   (`services/atlas-cashshop/.../inventory/asset/rest.go:8-57`) omit both
   fields, so atlas-channel structurally cannot see them.
3. The channel consequently hard-codes `GiftFrom: ""` when it projects locker
   assets, and never announces the gift list at all.

## Scope — two parts, both in scope

### F1 — the locker item's own sender field is blanked

`services/atlas-channel/.../socket/handler/cash_shop_entry.go:88` builds each
`cashcb.CashInventoryItem` with a literal `GiftFrom: ""` even though the asset
carries it. Populate it from the asset.

Note: the identical `GiftFrom: ""` literal in
`services/atlas-channel/.../kafka/consumer/cashshop/consumer.go`
(`handleStatusEventPurchase`, `handleStatusEventPackagePurchased`,
`handleStatusEventCouponRedeemed`) is **correct** — those are the buyer's own
just-purchased items, which have no sender. Leave them. Only the entry-burst
projection, which replays the whole locker including received gifts, is wrong.

### F2 — announce LOAD_GIFT_SUCCESS in the cash-shop entry burst

Build one `cashcb.GiftListEntry` per locker asset with a non-empty `GiftFrom`:
`{SN: asset.CashId(), ItemId: int32(asset.TemplateId()), BuyCharacterName:
asset.GiftFrom(), Text: asset.GiftMessage()}`, and announce it after the
existing `CashShopCashInventoryBody` call.

The packet layer is **already complete** — do not add to it:
- `cashcb.CashShopLoadGiftDoneBody(gifts []GiftListEntry)` exists at
  `libs/atlas-packet/cash/clientbound/shop_operation_body.go:547-550`.
- `LoadGiftDone` / `GiftListEntry` exist and match the v83 export read order
  (`CCashShop::OnCashItemResult#LOAD_GIFT_SUCCESS @ 0x47959e`:
  `Decode1(mode) · Decode2(count) · loop count DecodeBuffer(98B GW_GiftList)`;
  the 98-byte record is `liSN i64@0, nItemID i32@8, sBuyCharacterName
  char[13]@12, sText char[73]@25`).
- `LOAD_GIFT_SUCCESS` is bound in every template that has the arm: gms 61→49,
  72→57, 79→69, 83→77, 84→80, 87→82, 92→89, 95→90, jms_185→80.

**Version guard, required:** `template_gms_12_1.json` and
`template_gms_48_1.json` have a `CashShopOperation` writer but **no**
`LOAD_GIFT_SUCCESS` key (matching `docs/packets/dispatchers/cash_shop_operation.yaml:64-66`,
whose `modes` map starts at `gms_v61`). Announcing unconditionally would hit
`resolve.go`'s "not configured … defaulting to 99" path and push a garbage mode
at those clients. Gate the announce with
`atlas_packet.CodeConfigured(options, "operations", CashShopOperationLoadGiftDone)`
(`libs/atlas-packet/resolve.go:26`) — or, if the writer options are not reachable
from the handler, whatever the established idiom is for skipping an unbound arm.
Find that idiom rather than inventing one; if there is genuinely none, report
PARTIAL and say so instead of guessing.

## Fix — file inventory

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest.go`
  — add `GiftFrom string \`json:"giftFrom"\`` and `GiftMessage string
  \`json:"giftMessage"\`` to `RestModel`; carry both in `Transform` and
  `Extract`.
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/rest.go`,
  `model.go`, `builder.go` — carry the two fields through the channel-side
  model. Use the project's Builder pattern; no test-only constructors.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go`
  — F1 (populate `GiftFrom` at line 88) and F2 (the guarded
  `CashShopLoadGiftDoneBody` announce after the cash-inventory announce).
- Tests: a channel handler/codec test asserting the entry burst emits a
  `LOAD_GIFT_SUCCESS` body whose entries carry sender and message for a gifted
  asset and omit non-gift assets; an atlas-cashshop REST round-trip test
  (`Transform`→`Extract`) covering the two new fields. Follow the fixture style
  already in `kafka/consumer/cashshop/consumer_test.go`.
- `services/atlas-cashshop/docs/` — update the asset REST field list if one is
  documented there.

## Constraints

- Formatting authority is `tools/lint.sh` (no flags = fix mode). Bare `gofumpt`
  disagrees with the repo's golangci-lint fmt + goimports local-prefix config
  and will fail the gate.
- Verification scope is module-local (`go build` / `go test` for the two
  modules). A separate verifier owns `tools/verify.sh`.
- Defect G (ring pairing) is filed as its own GitHub issue and is **not** in
  scope. The unresolved gift-failure notice in the round-2 bug file is not in
  scope either.
- Never commit to `main`.
