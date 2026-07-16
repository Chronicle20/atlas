# SummonDamage (← `CSummonedPool::OnHit`)

- **IDA:** 0x6e9839
- **Atlas file:** `libs/atlas-packet/summon/clientbound/damage.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_848023@0x848062 (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `attackIdx (sub_6E9839@0x6e986a; atlas writes 12)` | ✅ |  |
| 3 | int32 | int32 `damage (@0x6e987f)` | ✅ |  |
| 4 | int32 | int32 `monsterIdFrom (@0x6e9891; read when attackIdx > -2)` | ✅ |  |
| 5 | byte | byte `bLeft (@0x6e989f; atlas writes 0)` | ✅ |  |

