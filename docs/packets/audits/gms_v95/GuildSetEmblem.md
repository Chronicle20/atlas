# GuildSetEmblem (← `CField::SendSetGuildMarkMsg`)

- **IDA:** 0x0
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_set_emblem.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `logoBackground` | ✅ |  |
| 1 | byte | byte `logoBackgroundColor` | ✅ |  |
| 2 | int16 | int16 `logo` | ✅ |  |
| 3 | byte | byte `logoColor` | ✅ |  |

