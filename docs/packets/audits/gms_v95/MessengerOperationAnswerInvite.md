# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x7f59d0
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 0 (ENTER) — messenger Operation mode byte` | ✅ |  |
| 1 | int32 | int32 `messengerId — room id passed in pData (the invite's room id)` | ✅ |  |

