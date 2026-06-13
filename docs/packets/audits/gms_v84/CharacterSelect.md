# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 0x60c1e3
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `` | ❌ | width mismatch |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | string | string `` | ✅ |  |
| 3 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | string `` | ❌ | atlas: short — missing trailing field |

