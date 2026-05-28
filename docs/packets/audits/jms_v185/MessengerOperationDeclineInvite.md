# MessengerOperationDeclineInvite (← `CFadeWnd::SendCloseMessage`)

- **IDA:** 0x557267
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_decline_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `sub-op = 5 (DECLINE)` | ❌ | width mismatch |
| 1 | string | string `fromName (inviter)` | ✅ |  |
| 2 | byte | string `myName (self)` | ❌ | width mismatch |
| 3 | byte | byte `pad = 0` | ❌ | atlas: short — missing trailing field |

