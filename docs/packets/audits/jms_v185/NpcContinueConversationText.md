# NpcContinueConversationText (← `CScriptMan::OnAskText#Reply`)

- **IDA:** 0x7b77bd
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/continue_conversation_text.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `msgType = 3 (AskText reply)` | ❌ | width mismatch |
| 1 | byte | byte `action (0=cancel, 1=text entered)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `answer text` | ❌ | atlas: short — missing trailing field |

