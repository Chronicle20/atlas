# FieldItcOperationCancelSale (← `CITC::OnCancelSaleItem`)

- **IDA:** 0x5624ec
- **Atlas file:** `../../libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7) @0x59fbdd` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59fbeb` | ✅ |  |

