# GuildForeignEmblemChanged (← `CUserRemote::OnGuildMarkChanged`)

- **IDA:** 0x0
- **Atlas file:** `libs/atlas-packet/guild/clientbound/emblem_changed_foreign.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (dispatcher-prefix pattern)` | ✅ |  |
| 1 | int16 | int16 `nMarkBg (logoBackground)` | ✅ |  |
| 2 | byte | byte `nMarkBgColor (logoBackgroundColor)` | ✅ |  |
| 3 | int16 | int16 `nMark (logo)` | ✅ |  |
| 4 | byte | byte `nMarkColor (logoColor)` | ✅ |  |

