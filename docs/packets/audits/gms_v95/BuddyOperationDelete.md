# BuddyOperationDelete (← `CField::SendDeleteFriendMsg`)

- **IDA:** 0x52f170
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_delete.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op (=3, DELETE) — sub-op byte for BUDDYLIST_MODIFY (opcode 153)` | ❌ | width mismatch |
| 1 | byte | int32 `dwFriendID — characterId of the buddy to remove` | ❌ | atlas: short — missing trailing field |

ack: OP-FAMILY-buddy tool-limitation — atlas-channel handler decodes BUDDYLIST_MODIFY in two steps: `buddy.Operation.Decode` reads the sub-op byte (op=3/DELETE) first; `OperationDelete.Decode` then reads friendId from the remaining bytes. Wire is correct. Deferred to _pending.md "OP-FAMILY-buddy" row.

