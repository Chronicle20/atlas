# GuildSetMemberTitle (← `CTabGuildAlliance::OnGradeChange`)

- **IDA:** 0x9ce1c7
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_member_title.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `new grade / alliance tier` | ❌ | width mismatch |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

