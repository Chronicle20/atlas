# GuildErrorMessageWithTarget (← `CWvsContext::OnGuildResult#ErrorMessageWithTarget`)

- **IDA:** 0xa37490
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x35/0x36/0x37)` | ✅ |  |
| 1 | string | string `targetName (v241)` | ✅ |  |

