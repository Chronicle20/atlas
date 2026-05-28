# MessengerAdd (← `CUIMessenger::OnPacket#Add`)

- **IDA:** 0x7f5e40
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/add.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0, ADD) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | byte | byte `position — slot index in messenger room (0–2)` | ✅ |  |
| 2 | byte | bytes `AvatarLook::AvatarLook(&v7, iPacket) — avatar appearance` | ❌ | width mismatch |
| 3 | byte | string `sID — character name` | ❌ | width mismatch |
| 4 | int32 | byte `channelId — channel the character is on` | ❌ | width mismatch |
| 5 | byte | byte `padding (extra flag, always discarded)` | ✅ |  |
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

