# BuddyOperationAccept (← `CField::SendAcceptFriendMsg`)

- **IDA:** 0x52f290
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_accept.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op (=2, ACCEPT) — sub-op byte for BUDDYLIST_MODIFY (opcode 153)` | ❌ | width mismatch |
| 1 | byte | int32 `dwFriendID — characterId of the buddy whose request is being accepted` | ❌ | atlas: short — missing trailing field |

