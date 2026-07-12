# FieldItcOperationRegisterWishEntry (← `CITC::OnRegisterWishEntry`)

- **IDA:** 0x5af8a5
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4) @0x5af92f` | ✅ |  |
| 1 | int32 | int32 `itemId @0x5af93d` | ✅ |  |
| 2 | int32 | int32 `price @0x5af94b` | ✅ |  |
| 3 | int32 | int32 `count @0x5af959` | ✅ |  |
| 4 | byte | byte `duration @0x5af96a` | ✅ |  |
| 5 | byte | byte `feeOption @0x5af97b` | ✅ |  |
| 6 | string | string `description @0x5af992` | ✅ |  |

