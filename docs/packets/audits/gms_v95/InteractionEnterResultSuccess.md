# InteractionEnterResultSuccess (← `CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccess`)

- **IDA:** 0x639500
- **Atlas file:** `../../libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (5; dispatch byte)` | ✅ |  |
| 1 | byte | bytes `room (roomType + maxUsers + myPosition + per-slot avatar loop; interaction.Room substruct)` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 50 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 51 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 52 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 53 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 54 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 55 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 56 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 57 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 58 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 59 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 60 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 61 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 62 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 63 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 64 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 65 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 66 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 67 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 68 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 69 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


> 🔍 tool limitation, NOT a wire bug. mode byte ✅; body is the shared
> interaction.Room sub-struct the analyzer cannot flatten. See
> `docs/packets/ida-exports/_pending.md` → "Interaction tool-limitation false positives".
