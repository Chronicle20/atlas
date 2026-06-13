# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0x9cb992
- **Atlas file:** `libs/atlas-packet/character/serverbound/move.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `fieldKey (Encode1; NO dr0/dr1/dr2/dr3/dwKey/crc32 in v83)` | ❌ | width mismatch |
| 1 | int32 | int32 `crc (field CRC for anti-cheat; GMS>28 guard still applies in v83)` | ✅ |  |
| 2 | byte | bytes `movement: CMovePath::Flush — encoded movement path; tool cannot linearize loop — ack:tool-limitation` | ✅ |  |
| 3 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

