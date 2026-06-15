# SummonDamageHandle (← `CSummoned::SetDamaged`)

- **IDA:** 0x828032
- **Atlas file:** `libs/atlas-packet/summon/serverbound/damage.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId ([this+44] = owner cid on jms185) — SetDamaged@0x828293` | ✅ |  |
| 1 | byte | byte `attackIdx (mob-present path; 0xFE sentinel on the no-mob path @0x8282a4) — SetDamaged@0x8282ba; atlas skip1` | ✅ |  |
| 2 | int32 | int32 `damage (nDamage) — SetDamaged@0x8282c3 (mob path) / @0x8282ad (no-mob path)` | ✅ |  |
| 3 | byte | int32 `mobTemplateId (monsterIdFrom; mob-present path only) — SetDamaged@0x8282e0` | ❌ | width mismatch |
| 4 | int32 | byte `dir<0 flag (mob-present path only) — SetDamaged@0x8282f0; atlas skip1. PRESENT on jms185.` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

