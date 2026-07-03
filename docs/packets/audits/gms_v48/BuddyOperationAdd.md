# BuddyOperationAdd (← `CField::SendSetFriendMsg`)

- **IDA:** 0x4c6452
- **Atlas file:** `libs/atlas-packet/buddy/serverbound/operation_add.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `BUDDYLIST_MODIFY mode = 1 (add) @0x4c6546` | ✅ |  |
| 1 | string | string `buddy character name @0x4c655f (v48 sends NO group name)` | ✅ |  |

