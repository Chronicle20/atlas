# GuildSetMemberTitle (← `CField::SendSetMemberGradeMsg`)

- **IDA:** 0x5585d1
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_member_title.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `targetId` | ✅ |  |
| 1 | byte | byte `newTitle byte` | ✅ |  |

