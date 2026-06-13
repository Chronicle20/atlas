# NpcAskSlideMenuConversationDetail (← `CScriptMan::OnAskSlideMenu#AskSlideMenu`)

- **IDA:** 0x76b5c8
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `menuType / dwType (this+33)` | ✅ |  |
| 1 | int32 | string `message text (v11)` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

