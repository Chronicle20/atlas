# EffectSkillUse (← `CUser::OnEffect`)

- **IDA:** 0x96ea92
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_skill_use.go`
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
| 3 | byte | byte `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 31 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 36 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 38 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 39 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 40 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 42 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 43 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

