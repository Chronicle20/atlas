# GuildSetMemberTitle (← `CField::SendSetMemberGradeMsg`)

- **IDA:** 0x56e1aa
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_member_title.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op=0xE` | ✅ |  |
| 1 | int32 | int32 `charId` | ✅ |  |
| 2 | byte | byte `grade` | ✅ |  |

