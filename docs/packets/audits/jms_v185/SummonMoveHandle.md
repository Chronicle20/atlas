# SummonMoveHandle (← `CVecCtrlSummoned::EndUpdateActive`)

- **IDA:** 0xaa5fc6
- **Atlas file:** `libs/atlas-packet/summon/serverbound/move.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (this[190] = owner cid on jms185) — CVecCtrlSummoned::EndUpdateActive@0xaa601e` | ✅ |  |
| 1 | bytes | bytes `movement blob — CMovePath::Flush@0xaa602b (opaque; head is Encode2 startX, Encode2 startY, Encode1 count, count moves, keypad run, bounding box). Atlas rebroadcasts the whole post-identity remainder byte-faithfully and extracts startX/startY from the first 4 bytes for position seeding.` | ✅ |  |

