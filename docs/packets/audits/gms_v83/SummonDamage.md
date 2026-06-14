# SummonDamage (← `CSummonedPool::OnHit`)

- **IDA:** 0x7a6ebe
- **Atlas file:** `libs/atlas-packet/summon/clientbound/damage.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by CSummonedPool::OnPacket@0x938dd7 before dispatch (pool is cid-keyed; NO oid on v83)` | ✅ |  |
| 1 | int32 | byte `attackIdx (signed) — CSummonedPool::OnSkill@0x7a6eef; atlas writes fixed 12` | ❌ | width mismatch |
| 2 | byte | int32 `damage (nDamage) — OnSkill@0x7a6efc` | ❌ | width mismatch |
| 3 | int32 | int32 `mobTemplateId (monsterIdFrom; only when attackIdx>-2) — OnSkill@0x7a6f0f (-> GetMobTemplate)` | ✅ |  |
| 4 | int32 | byte `bLeft (only when attackIdx>-2) — OnSkill@0x7a6f19; atlas writes fixed 0` | ❌ | width mismatch |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

