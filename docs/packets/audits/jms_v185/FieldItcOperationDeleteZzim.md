# FieldItcOperationDeleteZzim (← `CITC::OnDeleteZzim`)

- **IDA:** 0x604ed5
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xA) @0x604f01` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x604f12` | ✅ |  |

