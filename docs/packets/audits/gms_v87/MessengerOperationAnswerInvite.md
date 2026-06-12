# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x8b62ed
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 0 (ENTER) — messenger Operation mode byte` | ✅ |  |
| 1 | int32 | int32 `messengerId — room id` | ✅ |  |

