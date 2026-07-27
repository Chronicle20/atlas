# ChairFixed (← `CUserLocal::HandleXKeyDown`)

- **IDA:** 0x90a12b
- **Atlas file:** `libs/atlas-packet/character/serverbound/chair_fixed.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `chairId Encode2 @0x90a15f (get-up 0xFFFF); sit-down twin sub_85BF1A @0x85c028 emits COutPacket(41)+Encode2(seatIdx)` | ✅ |  |

