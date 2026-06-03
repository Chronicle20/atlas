# BuddyOperationAdd (← `CField::SendSetFriendMsg`)

- **IDA:** 0x56e41d
- **Atlas file:** `../../libs/atlas-packet/buddy/serverbound/operation_add.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `mode (sub-op: add/accept/delete)` | ❌ | width mismatch |
| 1 | string | string `target name` | ✅ |  |
| 2 | byte | string `group name` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `flag` | ❌ | atlas: short — missing trailing field |

