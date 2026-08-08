# MessengerOperationDeclineInvite (← `CFadeWnd::SendCloseMessage`)

- **IDA:** 0x4bce54
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_decline_invite.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (5 = DECLINE)` | ✅ |  |
| 1 | string | string `fromName` | ✅ |  |
| 2 | string | string `myName` | ✅ |  |
| 3 | byte | byte `trailing (0)` | ✅ |  |

