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


## Conditional secondary + detail body (tool limitation)

Rows 0–3 (the common header `speakerType` byte + `npcTemplate` int + `msgType`
byte + `param` byte) match the v83 client exactly. Rows 4–5 are artifacts of the
flat analyzer: atlas `NpcConversation.Encode` (conversation.go) writes a
**conditional** `secondaryNpcTemplateId` (`if param&4 { WriteInt }`) followed by
`WriteByteArray(conversationDetail)` (the per-dialog-type detail body). The
analyzer cannot model the conditional or the opaque byte-array sub-encoder, so it
flags them as "extra".

Verified against IDA `CScriptMan::OnScriptMessage@0x74660a`: the header is
`Decode1(speakerType)+Decode4(npcTemplate)+Decode1(msgType)+Decode1(bParam)`,
then `switch(msgType)` dispatches to a per-type handler. Each per-type handler
reads the `param&4` secondary template id FIRST (e.g. `CScriptMan::OnSay@0x7467ab`
line 0x7467cd: `if (a5&4) Decode4(secondary)`), then the detail body. Atlas draws
the wrapper/detail boundary at the same point — the secondary read and detail
body are correctly emitted by the per-type detail encoders (audited individually
in the NpcSay / NpcAsk* reports, all ✅). The `msgType` byte is resolved from
tenant config (`ResolveCode "messageType"`), so v83's shifted dialog-type enum
(no SayImage at index 1) is handled at the template/config layer, not the encoder.

**Verdict: ⚠️ (tool-limitation, manually verified — wire is correct for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
