# FieldItcOperationBuy (← `CITC::OnBuy`)

- **IDA:** 0x59f697
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x10) @0x59f6c3` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59f6d4` | ✅ |  |

