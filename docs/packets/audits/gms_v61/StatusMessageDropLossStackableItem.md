# StatusMessageDropLossStackableItem (← `CWvsContext::OnMessage#DropLossStackableItem`)

- **IDA:** 0x8438b5
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `drop type 0 @0x9192f4` | ✅ |  |
| 1 | byte | int32 `itemId @0x9193e7` | ❌ | width mismatch |
| 2 | int32 | int32 `amount (negative) @0x9193f4` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

