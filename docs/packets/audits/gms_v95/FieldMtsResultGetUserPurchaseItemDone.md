# FieldMtsResultGetUserPurchaseItemDone (← `CITC::OnNormalItemResult#GetUserPurchaseItemDone`)

- **IDA:** 0x576cf0
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation_list.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x21 GET_USER_PURCHASE_ITEM_DONE)` | ✅ |  |
| 1 | int32 | int32 `totalCount (loop count)` | ✅ |  |
| 2 | bytes | bytes `ITCITEM::Decode entry (repeated per loop count): opaque GW_ItemSlotBase blob (Decode1 type + RawDecode; model.Asset recurse) then Decode4 nITCSN/nPrice/nContractFee, DecodeStr txid/rollback, DecodeBuffer(8) ftExpired, DecodeStr user/game/comment, Decode4 nBidCount/Range/Price/Min/Max/Unit, Decode2 nProcessStatus. Whole entry opaque to the flat differ (MtsItem recurse collapses) — byte-fixture verified.` | ✅ |  |
| 3 | int32 | int32 `limitedCount` | ✅ |  |
| 4 | byte | byte `requestSent flag` | ✅ |  |

