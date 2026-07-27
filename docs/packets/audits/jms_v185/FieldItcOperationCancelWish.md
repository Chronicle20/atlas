# FieldItcOperationCancelWish (← `CITC::OnCancelWish`)

- **IDA:** 0x605059
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xD) @0x605085` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x605096` | ✅ |  |

