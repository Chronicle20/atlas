# FieldItcOperationSetZzim (← `CITC::OnSetZzim`)

- **IDA:** 0x5afc00
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x09) @0x5afc2c` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5afc3d` | ✅ |  |

