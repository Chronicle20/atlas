# FieldItcOperationCancelSale (← `CITC::OnCancelSaleItem`)

- **IDA:** 0x6050cc
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7) @0x60512f` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x60513d` | ✅ |  |

