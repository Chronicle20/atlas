# SummonDamage (← `CSummonedPool::OnHit`)

- **IDA:** 0x7598c0
- **Atlas file:** `libs/atlas-packet/summon/clientbound/damage.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId — read by CSummonedPool::OnPacket@0x75ac70 before dispatch` | ✅ |  |
| 1 | int32 | int32 `oid (dwSummonedID) — CUser::OnSummonedHit@0x8e3a42` | ✅ |  |
| 2 | byte | byte `attackIdx — CSummoned::OnHit@0x74bce2; atlas writes fixed 12` | ✅ |  |
| 3 | int32 | int32 `damage (nDamage) — CSummoned::OnHit@0x74bcf8` | ✅ |  |
| 4 | int32 | int32 `mobTemplateId (monsterIdFrom; only when attackIdx>-2) — CSummoned::OnHit@0x74bd0d` | ✅ |  |
| 5 | byte | byte `bLeft (only when attackIdx>-2) — CSummoned::OnHit@0x74bd1a; atlas writes fixed 0` | ✅ |  |

