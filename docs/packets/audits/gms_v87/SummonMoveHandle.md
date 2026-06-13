# SummonMoveHandle (← `CVecCtrlSummoned::EndUpdateActive`)

- **IDA:** 0xa591da
- **Atlas file:** `libs/atlas-packet/summon/serverbound/move.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (this[188] = owner cid on v87) — CVecCtrlSummoned::EndUpdateActive@0xa5922e` | ✅ |  |
| 1 | bytes | bytes `movement blob — CMovePath::Flush@0x6c74a1 (opaque; head is Encode2 startX, Encode2 startY, Encode1 count, count moves, keypad run, bounding box). Atlas rebroadcasts the whole post-identity remainder byte-faithfully and extracts startX/startY from the first 4 bytes for position seeding.` | ✅ |  |

