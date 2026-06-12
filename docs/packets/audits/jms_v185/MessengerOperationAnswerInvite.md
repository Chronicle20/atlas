# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x8e11b0
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 0 (ENTER)` | ✅ |  |
| 1 | int32 | int32 `messengerId` | ✅ |  |

