# FieldItcOperationBuy (← `CITC::OnBuy`)

- **IDA:** 0x5af9fa
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x10) @0x5afa26` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5afa37` | ✅ |  |

