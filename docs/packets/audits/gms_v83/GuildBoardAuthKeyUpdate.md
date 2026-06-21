# GuildBoardAuthKeyUpdate (← `CWvsContext::OnGuildResult#BoardAuthKeyUpdate`)

- **IDA:** 0xa37490
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (BOARD_AUTH_KEY_UPDATE)` | ✅ |  |
| 1 | string | string `authKey` | ✅ |  |

