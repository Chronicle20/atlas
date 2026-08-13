# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x61a701
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (0)` | ✅ |  |
| 1 | int32 | int32 `messengerId` | ✅ |  |

