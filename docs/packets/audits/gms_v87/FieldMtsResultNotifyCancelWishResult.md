# FieldMtsResultNotifyCancelWishResult (← `CITC::OnNormalItemResult#NotifyCancelWishResult`)

- **IDA:** 0x5d52f8
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_result_two_ints_modes.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x3D NotifyCancelWishResult)` | ✅ |  |
| 1 | int32 | int32 `Decode4 first notice count` | ✅ |  |
| 2 | int32 | int32 `Decode4 second notice count` | ✅ |  |

