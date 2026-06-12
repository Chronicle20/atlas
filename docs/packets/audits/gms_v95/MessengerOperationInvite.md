# MessengerOperationInvite (← `CUIMessenger::SendInviteMsg`)

- **IDA:** 0x7f5820
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 3 (INVITE) — messenger Operation mode byte` | ✅ |  |
| 1 | string | string `sTarget — target character name to invite` | ✅ |  |

