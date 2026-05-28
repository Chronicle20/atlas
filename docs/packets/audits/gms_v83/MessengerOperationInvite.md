# MessengerOperationInvite (← `CUIMessenger::SendInviteMsg`)

- **IDA:** 0x8511fc
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_invite.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `op (5)` | ❌ | width mismatch |
| 1 | byte | string `fromName` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `myName` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `pad (0)` | ❌ | atlas: short — missing trailing field |

