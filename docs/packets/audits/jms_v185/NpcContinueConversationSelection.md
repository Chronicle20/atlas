# NpcContinueConversationSelection (← `CScriptMan::OnAskMenu#Selection`)

- **IDA:** 0x7b7c95
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_selection.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `msgType = 5 (AskMenu @0x7b7d82)` | ❌ | width mismatch |
| 1 | byte | byte `action (@0x7b7da4)` | ✅ |  |
| 2 | byte | int32 `selection (only if action==1 @0x7b7daf)` | ❌ | atlas: short — missing trailing field |

