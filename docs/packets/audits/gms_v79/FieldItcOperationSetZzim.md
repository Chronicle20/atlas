# FieldItcOperationSetZzim (← `CITC::OnSetZzim`)

- **IDA:** 0x57ae3a
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (9) @0x59f8c9` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59f8da` | ✅ |  |

