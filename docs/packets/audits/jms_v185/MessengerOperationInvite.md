# MessengerOperationInvite (← `CUIMessenger::SendInviteMsg`)

- **IDA:** 0x8e4e8a
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 3 (INVITE)` | ✅ |  |
| 1 | string | string `target character name` | ✅ |  |

