# FieldItcOperationDeleteZzim (← `CITC::OnDeleteZzim`)

- **IDA:** 0x57af4b
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xA) @0x59f9da` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59f9eb` | ✅ |  |

