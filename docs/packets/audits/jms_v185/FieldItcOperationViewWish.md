# FieldItcOperationViewWish (← `CITC::OnViewWish`)

- **IDA:** 0x604f48
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xB) @0x604f74` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x604f85` | ✅ |  |

