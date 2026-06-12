# GuildSetEmblem (← `CField::SendSetGuildMarkMsg`)

- **IDA:** 0x52d8c0
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_set_emblem.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 15 (0xF, SET_MARK) — guild Operation mode byte` | ✅ |  |
| 1 | int16 | int16 `logoBackground (markBg)` | ✅ |  |
| 2 | byte | byte `logoBackgroundColor (markBgColor)` | ✅ |  |
| 3 | int16 | int16 `logo (mark)` | ✅ |  |
| 4 | byte | byte `logoColor (markColor)` | ✅ |  |

