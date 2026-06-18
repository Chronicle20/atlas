# FieldItcOperationBuyWish (← `CITC::OnBuyWish`)

- **IDA:** 0x604fbb
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xC) @0x605011` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x605022` | ✅ |  |

