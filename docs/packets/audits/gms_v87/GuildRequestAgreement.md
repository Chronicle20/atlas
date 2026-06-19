# GuildRequestAgreement (← `CWvsContext::OnGuildResult#RequestAgreement`)

- **IDA:** 0xacf7d3
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3)` | ✅ |  |
| 1 | int32 | int32 `guildId (check vs self)` | ✅ |  |
| 2 | string | string `character name 1` | ✅ |  |
| 3 | string | string `character name 2` | ✅ |  |

