# GuildSetMemberTitle (← `CField::SendSetMemberGradeMsg`)

- **IDA:** 0x4c61e4
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_member_title.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `GUILD_OPERATION mode = 0xE (SET_MEMBER_TITLE) @0x4c6206` | ✅ |  |
| 1 | int32 | int32 `target character id @0x4c6211` | ✅ |  |
| 2 | byte | byte `new grade/title @0x4c621c` | ✅ |  |

