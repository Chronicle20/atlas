# BuddyOperationDelete (← `CField::SendDeleteFriendMsg`)

- **IDA:** 0x56e5bd
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_delete.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `mode = 2 (delete)` | ❌ | width mismatch |
| 1 | byte | int32 `friendId` | ❌ | atlas: short — missing trailing field |

