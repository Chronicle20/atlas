# NpcNpcConversation (← `CScriptMan::OnScriptMessage`)

- **IDA:** 0x7b7160
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nSpeakerTypeID (@0x7b71a9)` | ✅ |  |
| 1 | int32 | int32 `nSpeakerTemplateID (@0x7b71b2)` | ✅ |  |
| 2 | byte | byte `nMsgType (dialog type; switch discriminator @0x7b71bf)` | ✅ |  |
| 3 | byte | byte `bParam (@0x7b71c7)` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, `CScriptMan::OnScriptMessage` @0x7b7160)

Rows 4-5 (❌ "atlas: extra") are an analyzer-flattening artifact, NOT a wire bug. The
wrapper reads `Decode1(nSpeakerTypeID) + Decode4(nSpeakerTemplateID) + Decode1(nMsgType) +
Decode1(bParam)` (rows 0-3 ✅). Atlas then writes the `secondaryNpcTemplateId` int only
when `param&4` (row 4) and the per-type `conversationDetail` byte array (row 5); both are
conditional/variable tails the analyzer collects unconditionally. The `param&4` secondary
int matches `CScriptMan::OnSay`'s `(bParam&4) Decode4` override, and the detail bytes are
consumed by the per-type handler selected by `nMsgType`. JMS185 switch order documented in
the IDA export (SayImage=1; NO AskMembershopAvatar case — case 9 is AskPet).

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
