# MessengerOperationAnswerInvite (← `CUIMessenger::OnCreate`)

- **IDA:** 0x7f59d0
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_answer_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `messengerId — room id passed in pData (the invite's room id); op byte (=0) stripped by atlas Operation dispatcher` | ✅ |  |

