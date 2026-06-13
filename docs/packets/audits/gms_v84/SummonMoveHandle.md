# SummonMoveHandle (← `CVecCtrlSummoned::EndUpdateActive`)

- **IDA:** 0xa0fd89
- **Atlas file:** `libs/atlas-packet/summon/serverbound/move.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (this[146] = ctrl[0x248] = owner cid on v84) — CVecCtrlSummoned::EndUpdateActive@0xa0fddd` | ✅ |  |
| 1 | bytes | bytes `movement blob — CMovePath__Flush@0xa0fdec (opaque; head is Encode2 startX, Encode2 startY, Encode1 count, count moves, keypad run, bounding box). Atlas rebroadcasts the whole post-identity remainder byte-faithfully and extracts startX/startY from the first 4 bytes for position seeding.` | ✅ |  |

