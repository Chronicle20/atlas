# SummonRemove (← `CSummonedPool::OnRemoved`)

- **IDA:** 0x75a470
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/remove.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId — read by CSummonedPool::OnPacket@0x75ac70 before dispatch` | ✅ |  |
| 1 | int32 | int32 `oid (dwSummonedID) — CSummonedPool::OnRemoved@0x75a4ae` | ✅ |  |
| 2 | byte | byte `animated flag (4=animated leave, 1=immediate) — consumed via CUser::OnSummonedRemoved@0x8e3790; atlas writes 4 or 1` | ✅ |  |

