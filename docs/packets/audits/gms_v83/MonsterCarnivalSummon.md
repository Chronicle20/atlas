# MonsterCarnivalSummon (← `CField_MonsterCarnival::OnRequestResult`)

- **IDA:** 0x56557d
- **Atlas file:** `libs/atlas-packet/monster/carnival/clientbound/monster_carnival_summon.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | string | byte `` | ❌ | width mismatch |
| 3 | byte | string `` | ❌ | atlas: short — missing trailing field |

