# StatusMessageForfeitQuestRecord (← `CWvsContext::OnMessage#ForfeitQuestRecord`)

- **IDA:** 0xb07e49
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (1 = quest record)` | ✅ |  |
| 1 | int16 | int16 `questId` | ✅ |  |
| 2 | byte | byte `inner disc byte = 0 (forfeit / remove quest); no follow-up bytes` | ✅ |  |

