# SummonDamageHandle (← `CSummoned::SetDamaged`)

- **IDA:** 0x7f879a
- **Atlas file:** `libs/atlas-packet/summon/serverbound/damage.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (this[43] = owner cid on v87) — SetDamaged@0x7f8a5c` | ✅ |  |
| 1 | byte | byte `attackIdx (mob-present path; 0xFE sentinel on the no-mob path @0x7f8a6d) — SetDamaged@0x7f8a83; atlas skip1` | ✅ |  |
| 2 | int32 | int32 `damage (nDamage) — SetDamaged@0x7f8a8c (mob path) / @0x7f8a76 (no-mob path)` | ✅ |  |
| 3 | byte | int32 `mobTemplateId (monsterIdFrom; mob-present path only) — SetDamaged@0x7f8aa9` | ❌ | width mismatch |
| 4 | int32 | byte `dir<0 flag (mob-present path only) — SetDamaged@0x7f8ab9; atlas skip1. PRESENT on v87 (not v95-only).` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

