# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x8511fc
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op (0)` | ❌ | width mismatch |
| 1 | byte | int32 `messengerId` | ❌ | atlas: short — missing trailing field |

