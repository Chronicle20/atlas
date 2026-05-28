# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x8e11b0
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `sub-op = 0 (ENTER)` | ❌ | width mismatch |
| 1 | byte | int32 `messengerId` | ❌ | atlas: short — missing trailing field |

