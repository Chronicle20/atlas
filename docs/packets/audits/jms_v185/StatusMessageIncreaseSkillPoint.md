# StatusMessageIncreaseSkillPoint (← `CWvsContext::OnMessage#IncreaseSkillPoint`)

- **IDA:** 0xb089ab
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (INCREASE_SKILL_POINT; v84+ only, v83 absent)` | ✅ |  |
| 1 | int16 | int16 `jobId` | ✅ |  |
| 2 | byte | byte `amount` | ✅ |  |

