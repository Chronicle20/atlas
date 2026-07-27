# StatusMessageGiveBuff (← `CWvsContext::OnMessage#GiveBuff`)

- **IDA:** 0x844971
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `itemId @0x91a6ca` | ❌ | width mismatch |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

