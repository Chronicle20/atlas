# BuddyOperationAdd (← `CField::SendSetFriendMsg`)

- **IDA:** 0x56e41d
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_add.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op=1` | ✅ |  |
| 1 | string | string `name` | ✅ |  |
| 2 | string | string `group` | ✅ |  |

