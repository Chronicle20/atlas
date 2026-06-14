# SummonAttack (← `CSummonedPool::OnAttack`)

- **IDA:** 0x828707
- **Atlas file:** `libs/atlas-packet/summon/clientbound/attack.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by CSummonedPool::OnPacket@0x9f7fad before dispatch (pool is cid-keyed; NO oid on jms185)` | ✅ |  |
| 1 | int32 | byte `charLevel (-> [this+188]) — CSummonedPool::OnAttack@0x82878d; atlas writes fixed 0` | ❌ | width mismatch |
| 2 | byte | byte `action byte (low7=action @0x8287ab, bit7=bLeft @0x8287a7) — OnAttack@0x82879b; atlas 'direction'` | ✅ |  |
| 3 | byte | byte `count (mob count) — OnAttack@0x8287db` | ✅ |  |
| 4 | byte | int32 `target[i].monsterOid — OnAttack@0x82880c, loop count times` | ❌ | width mismatch |
| 5 | int32 | byte `target[i].byte (only when monsterOid!=0) — OnAttack@0x82881a; atlas writes fixed 6` | ❌ | width mismatch |
| 6 | byte | int32 `target[i].damage (only when monsterOid!=0) — OnAttack@0x82882d` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

