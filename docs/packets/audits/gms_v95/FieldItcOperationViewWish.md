# FieldItcOperationViewWish (← `CITC::OnViewWish`)

- **IDA:** 0x5735c0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xB) @0x573608` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x57361c` | ✅ |  |

