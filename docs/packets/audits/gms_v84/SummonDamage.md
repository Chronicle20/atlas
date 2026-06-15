# SummonDamage (← `CSummonedPool::OnHit`)

- **IDA:** 0x7cc984
- **Atlas file:** `libs/atlas-packet/summon/clientbound/damage.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid — read by CSummonedPool::OnPacket@0x970237 before dispatch (pool is cid-keyed; NO oid on v84)` | ✅ |  |
| 1 | int32 | byte `attackIdx (signed) — sub_7CC984@0x7cc9b5; atlas writes fixed 12` | ❌ | width mismatch |
| 2 | byte | int32 `damage (nDamage) — sub_7CC984@0x7cc9ca` | ❌ | width mismatch |
| 3 | int32 | int32 `mobTemplateId (monsterIdFrom; only when attackIdx>-2) — sub_7CC984@0x7cc9dc (-> GetMobTemplate sub_6938FA)` | ✅ |  |
| 4 | int32 | byte `bLeft (only when attackIdx>-2) — sub_7CC984@0x7cc9ea; atlas writes fixed 0` | ❌ | width mismatch |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

