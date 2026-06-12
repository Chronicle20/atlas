# MessengerOperationInvite (← `CUIMessenger::SendInviteMsg`)

- **IDA:** 0x8511fc
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_invite.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 3 (INVITE)` | ✅ |  |
| 1 | string | string `target character name` | ✅ |  |

