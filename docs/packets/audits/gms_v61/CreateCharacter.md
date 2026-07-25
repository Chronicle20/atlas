# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x5653e9
- **Atlas file:** `libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name @0x56544c` | ✅ |  |
| 1 | int16 | int32 `face @0x565463 loop` | ❌ | width mismatch |
| 2 | int32 | int32 `hair` | ✅ |  |
| 3 | int32 | int32 `hairColor` | ✅ |  |
| 4 | int32 | int32 `skinColor` | ✅ |  |
| 5 | int32 | int32 `top` | ✅ |  |
| 6 | int32 | int32 `bottom` | ✅ |  |
| 7 | int32 | int32 `shoes` | ✅ |  |
| 8 | int32 | int32 `weapon` | ✅ |  |
| 9 | int32 | byte `gender @0x565482` | ❌ | width mismatch |
| 10 | byte | byte `strength @0x565495 loop` | ✅ |  |
| 11 | byte | byte `dexterity` | ✅ |  |
| 12 | byte | byte `intelligence` | ✅ |  |
| 13 | byte | byte `luck` | ✅ |  |
| 14 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

