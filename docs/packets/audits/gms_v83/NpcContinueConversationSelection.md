# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x746fad
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `selection / m_nSelect (AskMenu = 4-byte int; atlas ContinueConversationSelection wide path)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Runtime selection-width guard (tool limitation)

Row 0 (`int32` selection) matches the v83 wide-path client read. Row 1 (`byte`)
is an artifact of the flat analyzer: atlas
`ContinueConversationSelection.Encode` contains a **mutually-exclusive branch** —
`if m_wide { WriteInt32(selection) } else { WriteByte(selection) }` — and
`Decode` mirrors it via `if r.Available() >= 4 { ReadInt32; wide=true } else {
ReadByte }`. The analyzer inlines BOTH writes consecutively, producing the
spurious row-1 "extra".

The selection is the trailing field of the NPC_TALK_MORE reply, appended by the
dialog handler. (The leading `msgType`+`action` bytes belong to atlas
`ContinueConversation`, audited separately.) Width varies by dialog type,
verified against IDA:

| Dialog | IDA | Selection width |
|---|---|---|
| AskMenu (v83 msgType 4) | `CScriptMan::OnAskMenu@0x746fad` → `Encode4(selection)` @0x7470cf | 4 bytes (int32) |
| AskAvatar (v83 msgType 7) | `CScriptMan::OnAskAvatar@0x74713d` → `Encode1` @0x7472b0 | 1 byte |

Because the selection is the LAST field, atlas's `r.Available() >= 4` heuristic
correctly resolves to the wide path for AskMenu (4 trailing bytes) and the narrow
path for AskAvatar (1 trailing byte). Both real client widths are covered.

**Verdict: ⚠️ (tool-limitation, manually verified — both client widths covered for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
