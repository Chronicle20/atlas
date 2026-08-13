# MessengerOperationInvite (← `CUIMessenger::SendInviteMsg`)

- **IDA:** 0x61d8b8
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_invite.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 3 (INVITE)` | ✅ |  |
| 1 | string | string `target character name` | ✅ |  |

