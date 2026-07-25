# MonsterMonsterBomb (← `CMob::TryFirstSelfDestruction`)

- **IDA:** 0x5cd3fd
- **Atlas file:** `libs/atlas-packet/monster/serverbound/monster_bomb.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `COutPacket(160) self-destruct mobId` | ✅ |  |

