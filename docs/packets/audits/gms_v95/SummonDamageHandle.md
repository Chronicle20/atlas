# SummonDamageHandle (← `CSummoned::SetDamaged`)

- **IDA:** 0x74b730
- **Atlas file:** `libs/atlas-packet/summon/serverbound/damage.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `oid (m_dwSummonedID) — SetDamaged@0x74bb82` | ✅ |  |
| 1 | byte | byte `attackIdx (nAttackIdx) — SetDamaged@0x74bbae; atlas skip1` | ✅ |  |
| 2 | int32 | int32 `damage (nDamage) — SetDamaged@0x74bbb8` | ✅ |  |
| 3 | byte | int32 `mobTemplateId (monsterIdFrom) — SetDamaged@0x74bbd8` | ❌ | width mismatch |
| 4 | int32 | byte `dir<0 flag — SetDamaged@0x74bbed; atlas skip1 (v95+ DELTA)` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

