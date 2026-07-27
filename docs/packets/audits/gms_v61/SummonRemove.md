# SummonRemove (← `CSummonedPool::OnRemoved`)

- **IDA:** 0x7922e8
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/remove.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_7922E8 Decode4@0x792327 (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `leave/animated flag — sub_67BFED Decode1@0x67c002 (branched 0/2/3/4)` | ✅ |  |

