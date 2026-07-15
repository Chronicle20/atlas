# FieldItcOperationDeleteZzim (← `CITC::OnDeleteZzim`)

- **IDA:** 0x5afd11
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x0A) @0x5afd3d` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5afd4e` | ✅ |  |

