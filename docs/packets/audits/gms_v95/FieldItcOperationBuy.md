# FieldItcOperationBuy (← `CITC::OnBuy`)

- **IDA:** 0x573270
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x10) @0x5732b8` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5732cc` | ✅ |  |

