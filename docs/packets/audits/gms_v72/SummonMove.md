# SummonMove (← `CSummonedPool::OnMove`)

- **IDA:** 0x6e9285
- **Atlas file:** `libs/atlas-packet/summon/clientbound/move.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_848023@0x848062 (read before leaf dispatch)` | ✅ |  |
| 2 | bytes | bytes `raw CMovePath movement blob — sub_6E9285 -> CMovePath::OnMovePacket@0x635bc2` | ✅ |  |

