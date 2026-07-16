# BuddyOperationAccept (← `CField::SendAcceptFriendMsg`)

- **IDA:** 0x4c6643
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_accept.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `BUDDYLIST_MODIFY mode = 2 (accept) @0x4c66b7` | ✅ |  |
| 1 | int32 | int32 `fromCharacterId (inviter) @0x4c66c2` | ✅ |  |

