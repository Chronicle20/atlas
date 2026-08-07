# FieldItcOperationCancelSale (← `CITC::OnCancelSaleItem`)

- **IDA:** 0x59fb7a
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7) @0x59fbdd` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59fbeb` | ✅ |  |

