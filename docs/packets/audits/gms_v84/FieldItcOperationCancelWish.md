# FieldItcOperationCancelWish (← `CITC::OnCancelWish`)

- **IDA:** 0x5afe6a
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x0D) @0x5afe96` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5afea7` | ✅ |  |

