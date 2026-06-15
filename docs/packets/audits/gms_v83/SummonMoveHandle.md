# SummonMoveHandle (← `CVecCtrlSummoned::EndUpdateActive`)

- **IDA:** 0x9c84e9
- **Atlas file:** `libs/atlas-packet/summon/serverbound/move.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (ctrl[0x248] = owner cid on v83) — CVecCtrlSummoned::EndUpdateActive@0x9c853d` | ✅ |  |
| 1 | bytes | bytes `movement blob — CMovePath::Flush/Encode@0x68a563 (opaque; head is Encode2 startX, Encode2 startY, Encode1 count, count moves, keypad run, bounding box). Atlas rebroadcasts the whole post-identity remainder byte-faithfully and extracts startX/startY from the first 4 bytes for position seeding.` | ✅ |  |

