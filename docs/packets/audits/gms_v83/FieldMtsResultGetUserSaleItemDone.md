# FieldMtsResultGetUserSaleItemDone (← `CITC::OnNormalItemResult#GetUserSaleItemDone`)

- **IDA:** 0x5a4c57
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x23 GET_USER_SALE_ITEM_DONE)` | ✅ |  |
| 1 | int32 | int32 `totalCount (loop count)` | ✅ |  |
| 2 | bytes | bytes `ITCITEM::Decode entry (repeated per loop count): opaque GW_ItemSlotBase blob (Decode1 type + RawDecode; model.Asset recurse) then Decode4 nITCSN/nPrice/nContractFee, DecodeStr txid/rollback, DecodeBuffer(8) ftExpired, DecodeStr user/game/comment, Decode4 nBidCount/Range/Price/Min/Max/Unit, Decode2 nProcessStatus. Whole entry opaque to the flat differ (MtsItem recurse collapses) — byte-fixture verified.` | ✅ |  |

