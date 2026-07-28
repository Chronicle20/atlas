# TvTvSetMessage (← `CMapleTVMan::OnSetMessage`)

- **IDA:** 0x6371c1
- **Atlas file:** `libs/atlas-packet/tv/clientbound/set_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `flag (bit1 = receiverLook present)` | ✅ |  |
| 1 | byte | byte `messageType` | ✅ |  |
| 2 | bytes | bytes `AvatarLook::Decode(senderLook) — opaque avatar block (model.Avatar recurse)` | ✅ |  |
| 3 | string | string `senderName` | ✅ |  |
| 4 | string | string `receiverName` | ✅ |  |
| 5 | string | string `lines[0]` | ✅ |  |
| 6 | int32 | string `lines[1]` | ❌ | width mismatch |
| 7 | byte | string `lines[2]` | ❌ | width mismatch |
| 8 | byte | string `lines[3]` | ❌ | width mismatch |
| 9 | int32 | string `lines[4]` | ❌ | width mismatch |
| 10 | byte | int32 `totalWaitSeconds` | ❌ | width mismatch |
| 11 | int32 | bytes `AvatarLook::Decode(receiverLook) — opaque avatar block` | ✅ |  |
| 12 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

