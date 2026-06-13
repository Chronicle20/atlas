# SummonMove (← `CSummonedPool::OnMove`)

- **IDA:** 0x759830
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/move.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId — read by CSummonedPool::OnPacket@0x75ac70 before dispatch` | ✅ |  |
| 1 | int32 | int32 `oid (dwSummonedID) — CUser::OnSummonedMove@0x8e3892` | ✅ |  |
| 2 | int16 | int16 `startX — CMovePath::OnMovePacket movement-blob head` | ✅ |  |
| 3 | int16 | int16 `startY — CMovePath::OnMovePacket movement-blob head` | ✅ |  |
| 4 | bytes | bytes `rawMovement blob — CMovePath::OnMovePacket@0x6683f0 (variable-length movement path)` | ✅ |  |

