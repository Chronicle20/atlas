# StatusMessageDropLossUnStackableItem (← `CWvsContext::OnMessage#DropLossUnStackableItem`)

- **IDA:** 0x8438b5
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `drop type 2 @0x9192f4` | ✅ |  |
| 1 | byte | int32 `itemId @0x9194d9` | ❌ | width mismatch |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

