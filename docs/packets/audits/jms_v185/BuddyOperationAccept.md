# BuddyOperationAccept (← `CField::SendAcceptFriendMsg`)

- **IDA:** 0x56e66c
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_accept.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `mode = 1 (accept)` | ❌ | width mismatch |
| 1 | byte | int32 `friendId (inviter's character id)` | ❌ | atlas: short — missing trailing field |

