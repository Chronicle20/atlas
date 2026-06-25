# BuffGiveForeign (← `CUserRemote::OnSetTemporaryStat`)

- **IDA:** 0xa57431
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_give.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | bytes `stat mask (DecodeBuffer 16 = UINT128)` | ✅ |  |
| 1 | byte | byte `stat (mask bit)` | 🔍 | sub-struct: v — see _substruct/ |
| 2 | byte | byte `stat (mask bit)` | ✅ |  |
| 3 | byte | int32 `stat (mask bit)` | ❌ | width mismatch |
| 4 | byte | int32 `stat (mask bit)` | 🔍 | sub-struct: bts — see _substruct/ |
| 5 | int16 | int32 `stat (mask bit)` | ❌ | width mismatch |
| 6 | byte | int32 `stat (mask bit)` | ❌ | width mismatch |
| 7 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int16 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int16 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int16 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int32 `stat (mask bit)` | ❌ | atlas: short — missing trailing field |
| 29 | byte | byte `trailing nDefenseAtt (unconditional)` | ❌ | atlas: short — missing trailing field |
| 30 | byte | byte `trailing nDefenseState (unconditional)` | ❌ | atlas: short — missing trailing field |
| 31 | byte | unresolved `7x vtable-dispatched per-stat-set conditional decode (hand-trace)` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

