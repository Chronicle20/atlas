# NpcNpcConversation (← `CScriptMan::OnScriptMessage`)

- **IDA:** 0x74660a
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nSpeakerTypeID / v4 (speaker type)` | ✅ |  |
| 1 | int32 | int32 `nSpeakerTemplateID / a2a (npc template id)` | ✅ |  |
| 2 | byte | byte `nMsgType / v5 (dialog-type discriminator; v83 enum: 0=Say,1=AskYesNo,2=AskText,3=AskNumber,4=AskMenu,5=AskQuiz,6=AskSpeedQuiz,7=AskAvatar,8=AskMembershopAvatar,9=AskPet,10=AskPetAll,12=AskYesNoQuest,13=AskBoxText,14=AskSlideMenu — SHIFTED vs v95 which has SayImage=1)` | ✅ |  |
| 3 | byte | byte `bParam (speaker flags; bit 2 (param&4) gates a secondary npc template id read inside the per-type handler)` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |

