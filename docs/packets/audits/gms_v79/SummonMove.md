# SummonMove (← `CSummonedPool::OnMove`)

- **IDA:** 0x71cfc8
- **Atlas file:** `libs/atlas-packet/summon/clientbound/move.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket@0x8c8c84 (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_892500@0x89253f (read before leaf dispatch)` | ✅ |  |
| 2 | bytes | bytes `raw CMovePath movement blob — sub_71CFC8 -> CMovePath::OnMovePacket@0x6583fc (OPAQUE_LEDGER: begins with startX/startY, rebroadcast byte-faithfully)` | ✅ |  |

