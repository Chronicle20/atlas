# NpcNpcConversation (← `CScriptMan::OnScriptMessage`)

- **IDA:** 0x6de0f0
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nSpeakerTypeID (speaker type)` | ✅ |  |
| 1 | int32 | int32 `nSpeakerTemplateID (npc template id)` | ✅ |  |
| 2 | byte | byte `nMsgType (dialog-type discriminator: 0=Say,1=SayImage,2=AskYesNo,3=AskText,4=AskNumber,5=AskMenu,6=AskQuiz,7=AskSpeedQuiz,8=AskAvatar,9=AskMembershopAvatar,10=AskPet,11=AskPetAll,13=AskYesNoQuest,14=AskBoxText,15=AskSlideMenu)` | ✅ |  |
| 3 | byte | byte `bParam (speaker flags; bit 2 (param&4) gates a secondary npc template id read inside the per-type handler)` | ✅ |  |
| 4 | int32 | int32 `secondaryNpcTemplateId (read at start of per-type body when bParam&4; e.g. OnSay@0x6dc14b) -- guarded` | ✅ |  |
| 5 | bytes | bytes `per-type conversation detail body (opaque to wrapper; audited in NpcSay*/NpcAsk* reports) via WriteByteArray` | ✅ |  |

