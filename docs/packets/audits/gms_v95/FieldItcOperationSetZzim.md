# FieldItcOperationSetZzim (← `CITC::OnSetZzim`)

- **IDA:** 0x5733b0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (9) @0x5733f8` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x57340c` | ✅ |  |

