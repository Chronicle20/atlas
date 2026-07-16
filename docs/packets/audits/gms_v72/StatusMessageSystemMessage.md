# StatusMessageSystemMessage (← `CWvsContext::OnMessage#SystemMessage`)

- **IDA:** 0x919db7
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `message @0x919dc8` | ❌ | width mismatch |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |

