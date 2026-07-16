# FieldItcOperationBuyZzim (← `CITC::OnBuyZzim`)

- **IDA:** 0x573450
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x11) @0x5734ca` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5734de` | ✅ |  |

