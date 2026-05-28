# BuddyCapacityUpdate (← `CWvsContext::OnFriendResult#CapacityUpdate`)

- **IDA:** 0xb2a873
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/capacity_update.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op mode byte` | ✅ |  |
| 1 | byte | int32 `case 0x14: buddy count (CapacityUpdate)` | ❌ | width mismatch |
| 2 | byte | byte `case 0x15: new capacity byte` | ❌ | atlas: short — missing trailing field |

