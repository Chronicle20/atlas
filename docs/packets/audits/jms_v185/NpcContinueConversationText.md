# NpcContinueConversationText (← `CScriptMan::OnAskText#Reply`)

- **IDA:** 0x7b77bd
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_text.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `msgType = 3 (AskText @0x7b78d4)` | ❌ | width mismatch |
| 1 | byte | byte `action (@0x7b78e2)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `answer text (only if action==1 @0x7b7900)` | ❌ | atlas: short — missing trailing field |


## Manual verdict (JMS v185, `CScriptMan::OnAskText#Reply` @0x7b77bd)

The ❌ rows are a reply-builder alignment artifact, NOT a wire bug. The NPC_TALK_MORE
text-answer reply is built inside `CScriptMan::OnAskText` after `DoModal`:
`COutPacket(0x34) + Encode1(3=msgType) + Encode1(action) + (if action==1: EncodeStr answer)`.
Atlas splits the `msgType`+`action` prefix into the `ContinueConversation` wrapper and
`ContinueConversationText` carries only the conditional answer string. The audit aligns the
standalone answer against the full reply `calls`. The combined wrapper+text matches the
JMS185 reply byte-for-byte.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
