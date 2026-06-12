# BuddyOperationAdd (← `CField::SendSetFriendMsg`)

- **IDA:** 0x535240
- **Atlas file:** `../../libs/atlas-packet/buddy/serverbound/operation_add.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (=1, ADD) — sub-op byte for BUDDYLIST_MODIFY (opcode 153)` | ✅ |  |
| 1 | string | string `sTarget — name of the character to add as buddy` | ✅ |  |
| 2 | string | string `sFriendGroup — buddy group name` | ✅ |  |

