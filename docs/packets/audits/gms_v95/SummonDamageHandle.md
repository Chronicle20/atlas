# SummonDamageHandle (← `CSummoned::SetDamaged`)

- **IDA:** 0x74b730
- **Atlas file:** `../../libs/atlas-packet/summon/serverbound/damage.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `oid (m_dwSummonedID) — SetDamaged@0x74bb82` | ✅ |  |
| 1 | byte | byte `attackIdx (nAttackIdx) — SetDamaged@0x74bbae; atlas skip1` | ✅ |  |
| 2 | int32 | int32 `damage (nDamage) — SetDamaged@0x74bbb8` | ✅ |  |
| 3 | int32 | int32 `mobTemplateId (monsterIdFrom) — SetDamaged@0x74bbd8` | ✅ |  |
| 4 | byte | byte `dir<0 flag — SetDamaged@0x74bbed; atlas skip1 (v95+ DELTA)` | ✅ |  |

