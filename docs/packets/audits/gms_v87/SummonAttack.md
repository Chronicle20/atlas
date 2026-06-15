# SummonAttack (← `CSummonedPool::OnAttack`)

- **IDA:** 0x7f904c
- **Atlas file:** `libs/atlas-packet/summon/clientbound/attack.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by CSummonedPool::OnPacket@0x9b35f5 before dispatch (pool is cid-keyed; NO oid on v87)` | ✅ |  |
| 1 | int32 | byte `charLevel (m_nCharLevel -> [edi+0B8h]) — CSummonedPool::OnAttack@0x7f90d2; atlas writes fixed 0` | ❌ | width mismatch |
| 2 | byte | byte `action byte (low7=action @0x7f90f6, bit7=bLeft @0x7f90ea) — OnAttack@0x7f90e0; atlas 'direction'` | ✅ |  |
| 3 | byte | byte `count (mob count) — OnAttack@0x7f90fc` | ✅ |  |
| 4 | byte | int32 `target[i].monsterOid — OnAttack@0x7f9130, loop count times` | ❌ | width mismatch |
| 5 | int32 | byte `target[i].byte (only when monsterOid!=0) — OnAttack@0x7f913e; atlas writes fixed 6` | ❌ | width mismatch |
| 6 | byte | int32 `target[i].damage (only when monsterOid!=0) — OnAttack@0x7f914c` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

