# MonsterMobEscortFullPath (← `CMob::OnEscortFullPath`)

- **IDA:** 0x643d90
- **Atlas file:** `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | width mismatch |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |

