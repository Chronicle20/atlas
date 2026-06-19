# StatusMessageItemProtectExpire (← `CWvsContext::OnMessage#ItemProtectExpire`)

- **IDA:** 0xa6ccb3
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (ITEM_PROTECT_EXPIRE)` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | int32 | int32 `itemId (repeated count times)` | ✅ |  |

