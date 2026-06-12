# NpcNpcConversation (← `CScriptMan::OnScriptMessage`)

- **IDA:** 0x7b7160
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nSpeakerTypeID (@0x7b71a9)` | ✅ |  |
| 1 | int32 | int32 `nSpeakerTemplateID (@0x7b71b2)` | ✅ |  |
| 2 | byte | byte `nMsgType (dialog type; switch discriminator @0x7b71bf)` | ✅ |  |
| 3 | byte | byte `bParam (@0x7b71c7)` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |

