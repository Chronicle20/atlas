# BuddyOperationDelete (← `CField::SendDeleteFriendMsg`)

- **IDA:** 0x4c659b
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_delete.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `BUDDYLIST_MODIFY mode = 3 (delete) @0x4c6608` | ✅ |  |
| 1 | int32 | int32 `buddyCharacterId (target) @0x4c6613` | ✅ |  |

