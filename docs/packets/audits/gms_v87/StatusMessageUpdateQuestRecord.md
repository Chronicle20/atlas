# StatusMessageUpdateQuestRecord (← `CWvsContext::OnMessage#UpdateQuestRecord`)

- **IDA:** 0xab85d2
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (1 = quest record)` | ✅ |  |
| 1 | int16 | int16 `questId` | ✅ |  |
| 2 | byte | byte `inner disc byte = 1 (update)` | ✅ |  |
| 3 | string | string `quest info string` | ✅ |  |

