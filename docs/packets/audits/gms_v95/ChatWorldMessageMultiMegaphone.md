# ChatWorldMessageMultiMegaphone (← `CWvsContext::OnBroadcastMsg#MultiMegaphone`)

- **IDA:** 0xa04160
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `message[0]` | ✅ |  |
| 2 | byte | byte `count - case 10` | ✅ |  |
| 3 | string | string `message[1]` | ✅ |  |
| 4 | byte | string `message[2]` | ❌ | width mismatch |
| 5 | byte | byte `channelId - LABEL_31` | ✅ |  |
| 6 | byte | byte `whispersOn` | ❌ | atlas: short — missing trailing field |

