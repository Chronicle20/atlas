# ChatWhisper (← `CField::SendChatMsgWhisper`)

- **IDA:** 0x56bf11
- **Atlas file:** `libs/atlas-packet/chat/serverbound/whisper.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | string | string `` | ✅ |  |
| 3 | string | string `` | ✅ |  |
| 4 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | string `` | ❌ | atlas: short — missing trailing field |

