# DropSpawn (← `CDropPool::OnDropEnterField`)

- **IDA:** 0x50e789
- **Atlas file:** `libs/atlas-packet/drop/clientbound/spawn.go`
- **Variant:** GMS/v84
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
| 4 | int32 | int32 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int16 | int16 `` | ✅ |  |
| 7 | int16 | int16 `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int16 | int16 `` | ✅ |  |
| 10 | int16 | int16 `` | ✅ |  |
| 11 | int16 | int16 `` | ✅ |  |
| 12 | int64 | bytes `` | ✅ |  |
| 13 | byte | byte `` | ✅ |  |
| 14 | byte | byte `` | ❌ | atlas: short — missing trailing field |

