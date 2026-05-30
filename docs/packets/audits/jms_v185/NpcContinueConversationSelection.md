# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x7b7c95
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `msgType = 5 (AskMenu @0x7b7d82)` | ❌ | width mismatch |
| 1 | byte | byte `action (@0x7b7da4)` | ✅ |  |
| 2 | byte | int32 `selection (only if action==1 @0x7b7daf)` | ❌ | atlas: short — missing trailing field |


## Manual verdict (JMS v185, `CScriptMan::OnAskMenu#Selection` @0x7b7c95)

The ❌ rows are a reply-builder alignment artifact, NOT a wire bug. The NPC_TALK_MORE
menu-selection reply is built inside `CScriptMan::OnAskMenu` after `DoModal`:
`COutPacket(0x34) + Encode1(5=msgType) + Encode1(action) + (if action==1: Encode4 selection)`.
Atlas splits the `msgType` + `action` prefix into the `ContinueConversation` wrapper and
`ContinueConversationSelection` carries only the conditional `selection` (Int when wide, else
Byte). The audit aligns the standalone selection field against the full reply `calls`,
producing the apparent width/short mismatches. The combined wrapper+selection matches the
JMS185 reply byte-for-byte.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
