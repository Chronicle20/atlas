# SummonDamage (← `CSummonedPool::OnHit`)

- **IDA:** 0x67c936
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/damage.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_7922E8 Decode4@0x792327 (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `attackIdx — damage leaf sub_67C936 Decode1@0x67c967` | ✅ |  |
| 3 | int32 | int32 `damage — Decode4@0x67c97c` | ✅ |  |
| 4 | int32 | int32 `monsterIdFrom — Decode4@0x67c98e` | ✅ |  |
| 5 | byte | byte `bLeft — Decode1@0x67c99c` | ✅ |  |

