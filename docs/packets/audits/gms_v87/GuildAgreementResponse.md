# GuildAgreementResponse (← `CWvsContext::OnGuildResult#AgreementResponse`)

- **IDA:** 0xacf7d3
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `mode byte (37)` | ❌ | width mismatch |
| 1 | byte | int32 `guildId` | ❌ | width mismatch |
| 2 | byte | int32 `characterId` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `accepted (bool)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `characterName` | ❌ | atlas: short — missing trailing field |

