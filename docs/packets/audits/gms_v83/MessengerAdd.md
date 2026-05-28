# MessengerAdd (← `CUIMessenger::OnPacket#Add`)

- **IDA:** 0x8511fc
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/add.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

