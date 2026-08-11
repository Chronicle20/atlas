# InteractionOperationTransaction (← `CTradingRoomDlg::OnTrade`)

- **IDA:** 0x815773
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_transaction.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count (number of crc entries)` | ✅ |  |
| 1 | int32 | int32 `data/itemId (per-entry first)` | ✅ |  |
| 2 | int32 | int32 `crc (per-entry second)` | ✅ |  |

