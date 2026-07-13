# SummonDamage (← `CSummonedPool::OnHit`)

- **IDA:** 0x71d643
- **Atlas file:** `libs/atlas-packet/summon/clientbound/damage.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket@0x8c8c84 (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_892500@0x89253f (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `attackIdx (sub_71D643@0x71d674; atlas writes 12)` | ✅ |  |
| 3 | int32 | int32 `damage (@0x71d689)` | ✅ |  |
| 4 | int32 | int32 `monsterIdFrom -> GetMobTemplate (@0x71d69b)` | ✅ |  |
| 5 | byte | byte `bLeft (@0x71d6a9; atlas writes 0)` | ✅ |  |

