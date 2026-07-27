# FieldItcOperationRegisterWishEntry (← `CITC::OnRegisterWishEntry`)

- **IDA:** 0x604a69
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4) @0x604af3` | ✅ |  |
| 1 | int32 | int32 `itemId @0x604b01` | ✅ |  |
| 2 | int32 | int32 `price @0x604b0f` | ✅ |  |
| 3 | int32 | int32 `count @0x604b1d` | ✅ |  |
| 4 | byte | byte `duration @0x604b2e` | ✅ |  |
| 5 | byte | byte `feeOption @0x604b3f` | ✅ |  |
| 6 | string | string `description @0x604b56` | ✅ |  |

