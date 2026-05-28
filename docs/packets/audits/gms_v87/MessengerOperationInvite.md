# MessengerOperationInvite (← `CUIMessenger::SendInviteMsg`)

- **IDA:** 0x8b978f
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_invite.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `characterName to invite` | ✅ |  |

