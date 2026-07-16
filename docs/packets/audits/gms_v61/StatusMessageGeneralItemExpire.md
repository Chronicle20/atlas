# StatusMessageGeneralItemExpire (← `CWvsContext::OnMessage#GeneralItemExpire`)

- **IDA:** 0x844063
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count @0x919c12` | ✅ |  |
| 1 | byte | int32 `itemId (count loop) @0x919c30` | ❌ | width mismatch |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

