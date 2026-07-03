# StatusMessageUpdateQuestRecord (← `CWvsContext::OnMessage#UpdateQuestRecord`)

- **IDA:** 0x843bd8
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int16 `questId @0x919627` | ❌ | width mismatch |
| 1 | int16 | byte `subtype 1 @0x919638` | ❌ | width mismatch |
| 2 | byte | string `info @0x9196fa` | ❌ | width mismatch |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |

