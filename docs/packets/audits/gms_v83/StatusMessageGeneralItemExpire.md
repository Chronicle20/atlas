# StatusMessageGeneralItemExpire (← `CWvsContext::OnMessage#GeneralItemExpire`)

- **IDA:** 0xa217a2
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (GENERAL_ITEM_EXPIRE)` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | int32 | int32 `itemId (repeated count times)` | ✅ |  |

