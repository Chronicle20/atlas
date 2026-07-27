# FieldItcOperationCancelSale (← `CITC::OnCancelSaleItem`)

- **IDA:** 0x5737a0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7) @0x57382d` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x57383d` | ✅ |  |

