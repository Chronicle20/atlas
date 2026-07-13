# v79 Stage E — close reconciliation

Stage E's advance gate is "no **in-scope** cell remains ❌ (every in-scope cell is ✅ / 🟡-with-evidence / ⬜ / justified `_unimplemented.json` carve-out)". After the campaign + 5 closing batches, `go run ./tools/packet-audit matrix --check` is exit 0 (0 orphan/dangling/stale/drift, 0 v79 conflicts) and every existing version (v83/84/87/95/jms) is frozen. This note reconciles the residual matrix-❌ cells and shows **zero genuine unverified in-scope gaps**.

## Residual in-scope ❌ (12) — every one accounted for

The matrix still renders 12 in-scope cells as ❌. None is a real gap; they fall in two categories, both of which satisfy the gate:

### (A) op-row-verified artifacts (5) — packet IS verified
These packets are **verified via their op-row fixture**; the matrix additionally emits a `sub-struct` scoring row (for an embedded/opaque struct such as `model.Movement`) that is not independently scored, so it reads ❌ as benign duplication. The packet is covered.

| packet | op-row state | sub-struct row |
|---|---|---|
| character/serverbound/Move | ✅ verified | incomplete (opaque model.Movement, covered by op-row) |
| summon/serverbound/SummonAttackHandle | ✅ verified | incomplete (covered by op-row) |
| npc/serverbound/NpcStartConversation | ✅ verified | incomplete (covered by op-row) |
| character/serverbound/ExpressionRequest | ✅ verified | incomplete (covered by op-row) |
| field/serverbound/FieldChange | ✅ verified | incomplete (covered by op-row; CHANGE_MAP op-row ✅) |

### (B) justified `_unimplemented.json` dispositions (7) — v79-absent
These are genuinely absent from the v79 client and are documented in `docs/packets/audits/gms_v79/_unimplemented.json` (15 entries total) with a per-entry IDA basis. The matrix cannot currently render a `sub-struct` cell as n-a from `_unimplemented.json` (op-cells only), so they read ❌ despite the documented carve-out.

| packet | `_unimplemented.json` fname key | v79-absence basis (summary) |
|---|---|---|
| cash/…/CashShopOperationBuyNormal | (BuyNormal, escalation) | v79 mode 0x23 = 3 ints + 2 strings ≠ Atlas mode0x20/1-int; v79 serial-buy is OnBuyPackage |
| cash/…/CashShopOperationBuyWorldTransfer | CCashShop::SendBuyTransferWorldItemPacket | v79 OnBuyTransferWorldItem sends COutPacket(18) migrate, not CASHSHOP_OPERATION(221) |
| cash/…/CashShopOperationIncreaseStorage | CCashShop::OnIncTrunkCount | no trunk-count send in v79 CCashShop |
| cash/…/CashShopOperationMoveFromCashInventory | CCashShop::OnMoveCashItemLtoS | no locker→slot cash-move send in v79 |
| cash/…/CashShopOperationMoveToCashInventory | CCashShop::OnMoveCashItemStoL | no slot→locker cash-move send in v79 |
| cash/…/CashShopOperationRebateLockerItem | (rebate entry) | no rebate-locker send in v79 |
| npc/clientbound/NpcAskPetConversationDetail | CScriptMan::OnAskPet#AskPet | v79 has only the unified AskPetAll handler (case 10); no distinct single-pet body |

(Also in `_unimplemented.json`: 4 PIC character-select variants — v79 usesPic=false — and AskSlideMenu — no dispatcher case 14. These are login/npc cells already outside the residual-12 because their op-rows resolved.)

## Conclusion
- Every in-scope **tier-1 op-cell** is ✅.
- Every in-scope **packet** is verified (op-row) or a justified `_unimplemented.json` carve-out.
- **Zero silent/unexplained in-scope ❌.** The 12 residual matrix-❌ are 5 op-row-verified artifacts + 7 documented v79-absent dispositions.
- `matrix --check` exit 0; existing versions frozen.

## Known tooling limitation (follow-up, non-blocking)
The matrix scorer honors `_unimplemented.json` for `op` cells but not `sub-struct` cells, and emits a redundant `sub-struct` ❌ row for packets fully verified via their op-row. A conservative enhancement (render a `sub-struct` cell n-a **only** on an exact `_unimplemented.json` match, and treat an op-row-verified packet's embedded sub-struct as covered) would let the matrix render zero in-scope ❌ directly. Deferred deliberately — changing matrix scoring risks masking genuine ❌ across all versions (the "no false pass" bar), so it is left as a separate, test-guarded tooling task rather than folded into this pass.
