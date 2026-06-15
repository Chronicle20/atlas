# SummonDamageHandle (← `CSummoned::SetDamaged`)

- **IDA:** 0x7cbaf6
- **Atlas file:** `libs/atlas-packet/summon/serverbound/damage.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (this[43] = [obj+0xAC] = owner cid on v84) — SetDamaged@0x7cbd4d` | ✅ |  |
| 1 | byte | byte `attackIdx (mob-present path; 0xFE sentinel on the no-mob path @0x7cbd5e) — SetDamaged@0x7cbd74; atlas skip1` | ✅ |  |
| 2 | int32 | int32 `damage (nDamage) — SetDamaged@0x7cbd7d (mob path) / @0x7cbd67 (no-mob path)` | ✅ |  |
| 3 | byte | int32 `mobTemplateId (monsterIdFrom; mob-present path only) — SetDamaged@0x7cbd9a` | ❌ | width mismatch |
| 4 | int32 | byte `dir<0 flag (mob-present path only) — SetDamaged@0x7cbdaa; atlas skip1. PRESENT on v84 (not v95-only).` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

