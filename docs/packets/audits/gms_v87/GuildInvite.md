# GuildInvite (← `CWvsContext::OnGuildResult#Invite`)

- **IDA:** 0xacf7d3
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5)` | ✅ |  |
| 1 | int32 | int32 `guildId (v204)` | ✅ |  |
| 2 | string | string `inviterName (i)` | ✅ |  |
| 3 | int32 | int32 `unknown (v200) — present in v87 (NOT v83)` | ✅ |  |
| 4 | int32 | int32 `skillId (v201) — present in v87 (NOT v83)` | ✅ |  |

