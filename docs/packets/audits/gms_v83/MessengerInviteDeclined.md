# MessengerInviteDeclined (← `CUIMessenger::OnPacket#InviteDeclined`)

- **IDA:** 0x8511fc
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/invite_declined.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5)` | ✅ |  |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

