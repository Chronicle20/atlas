# FieldItcOperationViewWish (← `CITC::OnViewWish`)

- **IDA:** 0x5cf784
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x0B) @0x5cf7b0` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5cf7c1` | ✅ |  |

