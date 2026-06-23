# StatusMessageSkillExpire (← `CWvsContext::OnMessage#SkillExpire`)

- **IDA:** 0xb088a4
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (SKILL_EXPIRE)` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | int32 | int32 `skillId (repeated count times)` | ✅ |  |

