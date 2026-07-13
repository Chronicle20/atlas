# SummonMove (← `CSummonedPool::OnMove`)

- **IDA:** 0x67c37e
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/move.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_7922E8 Decode4@0x792327 (read before leaf dispatch)` | ✅ |  |
| 2 | bytes | bytes `raw CMovePath movement blob — sub_67C37E@0x67c37e -> CMovePath::OnMovePacket` | ✅ |  |

