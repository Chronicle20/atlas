# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x66e2ab
- **Atlas file:** `libs/atlas-packet/character/serverbound/create.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int16 | int32 `` | ❌ | width mismatch |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | string `` | ❌ | width mismatch |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | int32 | int16 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |

