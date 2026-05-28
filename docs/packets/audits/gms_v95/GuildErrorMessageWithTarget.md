# GuildErrorMessageWithTarget (← `CWvsContext::OnGuildResult#ErrorMessageWithTarget`)

- **IDA:** 0xa0d7d2
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (55=invite_sent, 56=invite_blocked, 57=invite_expired)` | ✅ |  |
| 1 | string | string `target character name` | ✅ |  |

