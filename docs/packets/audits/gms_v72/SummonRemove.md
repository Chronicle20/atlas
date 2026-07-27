# SummonRemove (← `CSummonedPool::OnRemoved`)

- **IDA:** 0x6e8f0f
- **Atlas file:** `libs/atlas-packet/summon/clientbound/remove.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_848023@0x848062 (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `leave/animated flag (sub_6E8F0F@0x6e8f24); oid read in dispatcher` | ✅ |  |

