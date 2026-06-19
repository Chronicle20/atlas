# GuildSetSkillResponse (← `CWvsContext::OnGuildResult#SetSkillResponse`)

- **IDA:** 0xa82e2b
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (SET_SKILL_RESPONSE)` | ✅ |  |
| 1 | byte | byte `success flag` | ✅ |  |
| 2 | string | string `message (success branch only)` | ✅ |  |

