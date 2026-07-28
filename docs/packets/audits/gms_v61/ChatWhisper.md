# ChatWhisper (← `sub_4E8635`)

- **IDA:** 0x4e8635
- **Atlas file:** `libs/atlas-packet/chat/serverbound/whisper.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `Encode1 mode=(!msgEmpty+1)\|4 @0x4e870a` | ✅ |  |
| 1 | int32 | string `EncodeStr targetName @0x4e8722` | ❌ | width mismatch |
| 2 | string | string `EncodeStr msg @0x4e8742` | ✅ |  |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |

