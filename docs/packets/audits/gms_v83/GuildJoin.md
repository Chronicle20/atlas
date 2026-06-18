# GuildJoin (← `CUIFadeYesNo::OnButtonClicked#Join`)

- **IDA:** 0x522585
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_join.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `guildId` | ✅ |  |
| 1 | int32 | int32 `characterId` | ✅ |  |

