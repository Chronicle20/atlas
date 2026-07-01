# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x5ccfa4
- **Atlas file:** `libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name @0x5cd029` | ✅ |  |
| 1 | int32 | int32 `jobIndex @0x5cd037` | ✅ |  |
| 2 | int16 | int32 `face @0x5cd048` | ❌ | width mismatch |
| 3 | int32 | int32 `hair @0x5cd048` | ✅ |  |
| 4 | int32 | int32 `hairColor @0x5cd048` | ✅ |  |
| 5 | int32 | int32 `skinColor @0x5cd048` | ✅ |  |
| 6 | int32 | int32 `top @0x5cd048` | ✅ |  |
| 7 | int32 | int32 `bottom @0x5cd048` | ✅ |  |
| 8 | int32 | int32 `shoes @0x5cd048` | ✅ |  |
| 9 | int32 | int32 `weapon @0x5cd048` | ✅ |  |
| 10 | int32 | byte `gender @0x5cd05f` | ❌ | width mismatch |
| 11 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

