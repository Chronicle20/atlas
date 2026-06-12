# MessengerInviteDeclined (← `CUIMessenger::OnPacket#InviteDeclined`)

- **IDA:** 0x8e4601
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/invite_declined.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `declineMode` | ✅ |  |

