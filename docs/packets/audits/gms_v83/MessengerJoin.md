# MessengerJoin (← `CUIMessenger::OnPacket#Join`)

- **IDA:** 0x8511fc
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/join.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (1)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

