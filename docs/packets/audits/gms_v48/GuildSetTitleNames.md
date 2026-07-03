# GuildSetTitleNames (← `CField::SendSetGradeNameMsg`)

- **IDA:** 0x4c624a
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_title_names.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `GUILD_OPERATION mode = 0xD (SET_TITLE_NAMES) @0x4c6275` | ✅ |  |
| 1 | string | string `grade title 1 @0x4c628e` | ✅ |  |
| 2 | string | string `grade title 2 @0x4c62a7` | ✅ |  |
| 3 | string | string `grade title 3 @0x4c62c0` | ✅ |  |
| 4 | string | string `grade title 4 @0x4c62d9` | ✅ |  |
| 5 | string | string `grade title 5 @0x4c62f2` | ✅ |  |

