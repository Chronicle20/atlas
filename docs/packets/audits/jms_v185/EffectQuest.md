# EffectQuest (← `CUser::OnEffect`)

- **IDA:** 0x9f6395
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_quest.go`
- **Variant:** JMS/v185
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
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 36 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 38 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 39 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 40 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 42 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 43 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 44 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 45 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 46 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

