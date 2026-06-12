# MessengerOperationDeclineInvite (← `CFadeWnd::SendCloseMessage`)

- **IDA:** 0x557267
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_decline_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 5 (DECLINE)` | ✅ |  |
| 1 | string | string `fromName (inviter)` | ✅ |  |
| 2 | string | string `myName (self)` | ✅ |  |
| 3 | byte | byte `pad = 0` | ✅ |  |

