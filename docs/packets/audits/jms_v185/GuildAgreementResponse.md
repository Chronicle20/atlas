# GuildAgreementResponse (← `CWvsContext::OnGuildResult#AgreementResponse`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `mode (AgreementResponse/RequestAgreement)` | ❌ | width mismatch |
| 1 | byte | int32 `partyId/guildId` | ❌ | width mismatch |
| 2 | byte | string `leaderName` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `guildName` | ❌ | atlas: short — missing trailing field |

