# FieldItcOperationDeleteZzim (← `CITC::OnDeleteZzim`)

- **IDA:** 0x573520
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xA) @0x573568` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x57357c` | ✅ |  |

