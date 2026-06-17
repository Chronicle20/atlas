# FieldMtsResultGetSearchItcListDone (← `CITC::OnNormalItemResult#GetSearchItcListDone`)

- **IDA:** 0x5d4af2
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x17 GET_SEARCH_ITC_LIST_DONE)` | ✅ |  |
| 1 | int32 | int32 `categoryItemCnt` | ✅ |  |
| 2 | int32 | int32 `pageItemCnt (loop count)` | ✅ |  |
| 3 | int32 | int32 `category` | ✅ |  |
| 4 | int32 | int32 `subCategory` | ✅ |  |
| 5 | int32 | int32 `page` | ✅ |  |
| 6 | bytes | bytes `ITCITEM::Decode entry (repeated per loop count): opaque GW_ItemSlotBase blob (Decode1 type + RawDecode; model.Asset recurse) then Decode4 nITCSN/nPrice/nContractFee, DecodeStr txid/rollback, DecodeBuffer(8) ftExpired, DecodeStr user/game/comment, Decode4 nBidCount/Range/Price/Min/Max/Unit, Decode2 nProcessStatus. Whole entry opaque to the flat differ (MtsItem recurse collapses) — byte-fixture verified.` | ✅ |  |

