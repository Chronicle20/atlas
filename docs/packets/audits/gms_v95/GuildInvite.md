# GuildInvite (← `CWvsContext::OnGuildResult#Invite`)

- **IDA:** 0xa0d664
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | string | string `inviterName` | ✅ |  |
| 3 | int32 | int32 `v21 (unknown — atlas MISSING this field)` | ✅ |  |
| 4 | int32 | int32 `nSkillID (unknown — atlas MISSING this field)` | ✅ |  |

