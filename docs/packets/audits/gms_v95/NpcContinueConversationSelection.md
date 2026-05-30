# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x6dce00
- **Atlas file:** `libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `selection (m_nSelect; AskMenu = 4-byte int; atlas ContinueConversationSelection wide path)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

## Runtime selection-width guard (tool limitation)

Row 0 matches the v95 client exactly (`int32` selection). Row 1 (`byte`) is an
artifact of the flat analyzer: atlas `ContinueConversationSelection.Encode`
contains a **mutually-exclusive branch** — `if m_wide { WriteInt32(selection) }
else { WriteByte(selection) }` — and `Decode` mirrors it via `if r.Available()
>= 4 { ReadInt32; wide=true } else { ReadByte; wide=false }`. The analyzer cannot
model the branch direction, so it inlines BOTH the `int32` (wide) and the `byte`
(narrow) writes consecutively, producing the spurious row-1 "extra".

The selection field is part of the NPC_TALK_MORE reply (opcode 65/0x41) appended
by the dialog handler when the user confirms a choice. Its width varies by dialog
type, verified against IDA:

| Dialog | IDA | Selection width |
|---|---|---|
| AskMenu (msgType 5) | `CScriptMan::OnAskMenu@0x6dce00` → `Encode4(m_nSelect)` @0x6dcf5f | 4 bytes (int32) |
| AskAvatar (msgType 8) | `CScriptMan::OnAskAvatar@0x6dcff0` → `Encode1(m_nAvatarIndex)` @0x6dd26e | 1 byte |

Because the selection is the LAST field in the packet body (after the
dispatcher-consumed `msgType`+`action` bytes read by `ContinueConversation`),
atlas's `r.Available() >= 4` heuristic correctly resolves to the wide path for
AskMenu (4 trailing bytes) and the narrow path for AskAvatar (1 trailing byte).
Both real client widths are covered by the wide/narrow branch.

**Verdict: ⚠️ (tool-limitation, manually verified — both client widths covered).**


Ack: world-audit Phase 2g on 2026-05-28
