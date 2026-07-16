# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x646e33
- **Atlas file:** `libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** GMS/v79
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | bytes | bytes `` | ✅ |  |
| 4 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

