# StatusMessageQuestRecordEx (← `CWvsContext::OnMessage#QuestRecordEx`)

- **IDA:** 0x919a7b
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int16 `questId @0x919a98` | ❌ | width mismatch |
| 1 | int16 | string `info @0x919a9e` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

