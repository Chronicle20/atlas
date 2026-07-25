# MonsterControl (← `CMobPool::OnMobChangeController`)

- **IDA:** 0x647150
- **Atlas file:** `libs/atlas-packet/monster/clientbound/control.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | bytes | bytes `` | ✅ |  |
| 5 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

