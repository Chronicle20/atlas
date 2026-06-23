# StatusMessageCashItemExpire (← `CWvsContext::OnMessage#CashItemExpire`)

- **IDA:** 0xa6cb31
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (CASH_ITEM_EXPIRE)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |

