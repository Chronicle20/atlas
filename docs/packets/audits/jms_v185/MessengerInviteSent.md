# MessengerInviteSent (← `CUIMessenger::OnPacket#InviteSent`)

- **IDA:** 0x8e4515
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/invite_sent.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 4 (InviteSent/OnInviteResult)` | ✅ |  |
| 1 | string | string `msg` | ✅ |  |
| 2 | byte | byte `success flag` | ✅ |  |

