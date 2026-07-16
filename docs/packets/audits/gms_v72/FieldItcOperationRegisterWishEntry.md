# FieldItcOperationRegisterWishEntry (← `CITC::OnRegisterWishEntry`)

- **IDA:** 0x561eb4
- **Atlas file:** `../../libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4) @0x59f5cc` | ✅ |  |
| 1 | int32 | int32 `itemId @0x59f5da` | ✅ |  |
| 2 | int32 | int32 `price @0x59f5e8` | ✅ |  |
| 3 | int32 | int32 `count @0x59f5f6` | ✅ |  |
| 4 | byte | byte `duration @0x59f607` | ✅ |  |
| 5 | byte | byte `feeOption @0x59f618` | ✅ |  |
| 6 | string | string `description @0x59f62f` | ✅ |  |

