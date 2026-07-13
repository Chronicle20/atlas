# EffectQuest (← `CUser::OnEffect`)

- **IDA:** 0x79148d
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_quest.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | int32 `` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

