# InteractionOperationTradePutItem (← `CTradingRoomDlg::PutItem`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_trade_put_item.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

