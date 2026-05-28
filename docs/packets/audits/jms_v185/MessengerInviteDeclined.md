# MessengerInviteDeclined (← `CUIMessenger::OnPacket#InviteDeclined`)

- **IDA:** 0x8e4601
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/invite_declined.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 5 (InviteDeclined/OnBlocked)` | ✅ |  |
| 1 | string | string `blocked user name` | ✅ |  |
| 2 | byte | byte `declineMode` | ✅ |  |

