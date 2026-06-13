# SummonMoveHandle (← `CVecCtrlSummoned::EndUpdateActive`)

- **IDA:** 0x9a0700
- **Atlas file:** `../../libs/atlas-packet/summon/serverbound/move.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `oid (m_dwSummonedID) — CVecCtrlSummoned::EndUpdateActive@0x9a0775` | ✅ |  |
| 1 | int16 | int16 `startX — CMovePath::Flush movement-blob head` | ✅ |  |
| 2 | int16 | int16 `startY — CMovePath::Flush movement-blob head` | ✅ |  |
| 3 | bytes | bytes `rawMovement blob — CMovePath::Flush@0x668160 (variable-length movement path)` | ✅ |  |

