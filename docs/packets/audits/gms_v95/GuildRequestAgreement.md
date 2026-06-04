# GuildRequestAgreement (← `CWvsContext::OnGuildResult#RequestAgreement`)

- **IDA:** 0xa0d3b0
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | string | string `leaderName (gradeName in IDA)` | ✅ |  |
| 3 | string | string `guildName (skillID string in IDA)` | ✅ |  |

