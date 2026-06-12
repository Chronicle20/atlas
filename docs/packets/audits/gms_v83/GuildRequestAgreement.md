# GuildRequestAgreement (← `CWvsContext::OnGuildResult#AgreementResponse`)

- **IDA:** 0xa37490
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | string | string `leaderName` | ✅ |  |
| 3 | string | string `guildName` | ✅ |  |

