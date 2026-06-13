# BuddyCapacityUpdate (← `CWvsContext::OnFriendResult#CapacityUpdate`)

- **IDA:** 0xad7ae5
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/capacity_update.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x15)` | ✅ |  |
| 1 | byte | byte `maxFriendCount (new capacity)` | ✅ |  |

