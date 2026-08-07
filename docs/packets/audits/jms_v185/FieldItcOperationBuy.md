# FieldItcOperationBuy (← `CITC::OnBuy`)

- **IDA:** 0x604bbe
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x10) @0x604bea` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x604bfb` | ✅ |  |

