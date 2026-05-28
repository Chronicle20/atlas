# BuddyCapacityUpdate (← `CWvsContext::OnFriendResult#CapacityUpdate`)

- **IDA:** 0xa12630
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/capacity_update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0x15/21, CAPACITY_UPDATE) — dispatcher switch byte consumed by OnFriendResult` | ✅ |  |
| 1 | byte | byte `nFriendMax — new buddy-list capacity written into CharacterData.nFriendMax` | ✅ |  |

