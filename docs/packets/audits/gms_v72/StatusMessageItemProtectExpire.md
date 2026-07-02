# StatusMessageItemProtectExpire (← `CWvsContext::OnMessage#ItemProtectExpire`)

- **IDA:** 0x919cdb
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count @0x919cee` | ✅ |  |
| 1 | byte | int32 `itemId (count loop) @0x919d0c` | ❌ | width mismatch |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

