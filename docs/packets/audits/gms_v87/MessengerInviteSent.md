# MessengerInviteSent (← `CUIMessenger::OnPacket#InviteSent`)

- **IDA:** 0x8b978f
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/invite_sent.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4)` | ✅ |  |
| 1 | string | byte `result (0=OK, 1=blocked, 2=full)` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

