# StatusMessageItemExpireReplace (← `CWvsContext::OnMessage#ItemExpireReplace`)

- **IDA:** 0xab8fec
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (ITEM_EXPIRE_REPLACE)` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | string | string `message (repeated count times)` | ✅ |  |

